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

type BotCommand func(b *IRCBot, input string)

type IRCBot struct {
	config IRCBotConfig
	connection *tls.Conn
	stream chan string
	interruptSignal chan os.Signal
	handlers map[string]BotCommand
}

func NewIRCBot(config IRCBotConfig) *IRCBot {
	return &IRCBot{
		config: config,
		connection: nil,
		stream: nil,
		interruptSignal: nil,
		handlers: make(map[string]BotCommand),
	}
}

func (bot *IRCBot) AddBotCommand(command string, fn BotCommand){
	bot.handlers[command] = fn
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
			log.Printf("RECV: %s\n", line);
			if err != nil {
				panic(err)
			}
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
	fmt.Fprintf(bot.connection, "QUIT %s\r\n", bot.config.QuitMessage)
}

func (bot *IRCBot) Serve() {
	defer bot.connection.Close()
	firstPing := true
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
				break
			}


			if strings.HasPrefix(msg.Params[len(msg.Params) - 1], "!echo") {
				fmt.Fprintf(bot.connection, "PRIVMSG #test :%q\r\n", msg.Params[len(msg.Params) - 1])
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
