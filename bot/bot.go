package bot

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net/textproto"
	"os"
	"os/signal"
	"syscall"
	"strings"
	message "github.com/antonio-petrillo/ircbot/message"
)

type IRCBotConfig struct {
	Host        string
	Port        string
	TlsConfig   *tls.Config
	Password    string
	Nickname    string
	Username    string
	QuitMessage string
}

type CommandAction func(b *IRCBot, msg *message.Message)
type ParamsAction func(b *IRCBot, msg *message.Message)

type IRCBot struct {
	config          IRCBotConfig
	connection      *tls.Conn
	stream          chan string
	interruptSignal chan os.Signal
	commandHandlers map[string]CommandAction
	paramsHandlers  map[string]ParamsAction
}

func NewIRCBot(config IRCBotConfig) *IRCBot {
	return &IRCBot{
		config: config,
		connection: nil,
		stream: nil,
		interruptSignal: nil,
		commandHandlers: make(map[string]CommandAction),
		paramsHandlers: make(map[string]ParamsAction),
	}
}

func (bot *IRCBot) AddCommandHandler(command string, fn CommandAction) {
	bot.commandHandlers[command] = fn
}

func (bot *IRCBot) AddParamsHandler(suffix string, fn ParamsAction) {
	bot.paramsHandlers[suffix] = fn
}

func (bot *IRCBot) Connect() {
	addr := bot.config.Host + ":" + bot.config.Port
	conn, err := tls.Dial("tcp", addr, bot.config.TlsConfig)
	if err != nil {
		log.Fatalf("[%s] Cannot open tcp connection\n", "CONNECTION")
	}

	bot.connection = conn
	bot.stream = make(chan string)

	bot.interruptSignal = make(chan os.Signal)
	signal.Notify(bot.interruptSignal, os.Interrupt, syscall.SIGTERM)
	signal.Notify(bot.interruptSignal, os.Interrupt, syscall.SIGINT)

	go func() {
		textProtocol := textproto.NewReader(bufio.NewReader(conn))
		stream := bot.stream
		for {
			line, err := textProtocol.ReadLine()
			if err != nil {
				close(stream)
			}
			log.Printf("RECV: %s\n", line);
			stream <- line
		}
	} ()

	go func() {
		interrupter := bot.interruptSignal
		<-interrupter
		bot.Disconnect()
	} ()
}

func (bot *IRCBot) Login() {
	if bot.config.Password != "" {
		log.Printf("PASS %s", bot.config.Password)
		fmt.Fprintf(bot.connection, "PASS %s\r\n", bot.config.Password)
	}

	fmt.Fprintf(bot.connection, "NICK %s\r\n", bot.config.Nickname)
	log.Printf("NICK %s", bot.config.Nickname)
	fmt.Fprintf(bot.connection, "USER %s 8 * %s\r\n", bot.config.Username, bot.config.Username)
	log.Printf("USER %s", bot.config.Username)
}

func (bot *IRCBot) Disconnect() {
	log.Printf("QUIT %s\n", bot.config.QuitMessage)
	fmt.Fprintf(bot.connection, "QUIT %s\r\n", bot.config.QuitMessage)
}

func (bot *IRCBot) Serve() {
	defer bot.connection.Close()
	firstPing := true

	backlog := []string{}
loop:
	for {
		select {
		case line := <- bot.stream:
			//check for command
			log.Printf("RECV: %s\n", line)
			msg, err := message.ParseMessage(line)
			if err != nil {
				log.Printf("Parse Error: %q\n", err)
				continue
			}

			if msg.Command == "PING" {
				fmt.Fprintf(bot.connection, "PONG %s\r\n", line[5:])
				if firstPing {
					fmt.Fprintf(bot.connection, "JOIN #test\r\n")
					firstPing = false
				}
				continue
			}

			///////////////////////////////
			// HACK to implement history //
			///////////////////////////////
			if msg.Command == "PRIVMSG" && len(msg.Params) > 0 {
				trailing := msg.Params[len(msg.Params) - 1]
				if trailing != "" && !strings.HasPrefix(trailing, "!history") {
					backlog = append(backlog, trailing)
				}

				if strings.HasPrefix(trailing, "!history") {
					go func() {
						if len(backlog) > 0 {
							fmt.Fprintf(bot.connection, "PRIVMSG #test here the previous %d messages\r\n", len(backlog))
							for i, oldMessage := range backlog {
								fmt.Fprintf(bot.connection, "PRIVMSG #test history[%d]=%s\r\n", i, oldMessage)
							}
						} else {
							fmt.Fprintf(bot.connection, "PRIVMSG #test no message to retrieve\r\n")
						}
					} ()
				}
				continue
			}



			///////////////////////////////

			if fn, ok := bot.commandHandlers[msg.Command]; ok {
				go func() {
					fn(bot, msg)
				}()
				continue
			}

			size := len(msg.Params)
			if size > 0 && len(msg.Params[size - 1]) > 0 {
				// allow params action only on suffix on last params
				key := strings.Split(msg.Params[size - 1], " ")[0]
				if fn, ok := bot.paramsHandlers[key]; ok {
					go func() {
						fn(bot, msg)
					}()
				}
				continue
			}
		case <-bot.interruptSignal:
			log.Printf("IRCBOT %s (%s) quit\n", bot.config.Nickname, bot.config.Username)
			fmt.Fprintf(bot.connection, "QUIT %s\r\n", bot.config.QuitMessage)
			close(bot.stream)
			break loop
		}
	}
}

func (bot *IRCBot) SendMsg(command string) {
	fmt.Fprintf(bot.connection, command)
}
