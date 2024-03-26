package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"

	irc "github.com/antonio-petrillo/ircbot/bot"
	message "github.com/antonio-petrillo/ircbot/message"
)

func main() {
	config := irc.IRCBotConfig{
		Host: "irc.someserver.com",
		Port: "6697",
		TlsConfig: &tls.Config{
			// self accecpt certificate (can be done better)
			InsecureSkipVerify: true,
		},
		Password: "your_password",
		Nickname: "your_nick",
		Username: "your_username",
		QuitMessage: "Goodbye Cruel World",
	}

	bot := irc.NewIRCBot(config)

	bot.AddParamsHandler("!echo", func(bot *irc.IRCBot, msg *message.Message) {
		size := len(msg.Params)
		echo := msg.Params[size - 1]
		log.Printf("params: %v\n", msg.Params)
		params := ""
		if (len(msg.Params) > 1) {
			params = strings.Join(msg.Params[0:size - 1], " ")
		}
		message := fmt.Sprintf("%s %s %q\r\n", msg.Command, params, echo)
		log.Printf("Echo message: %s\n", message)
		bot.SendMsg(message)
	})

	bot.Connect()
	bot.Login()
	bot.Serve()
}
