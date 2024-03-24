package main

import (
	"crypto/tls"
	"fmt"
	irc "github.com/antonio-petrillo/ircbot/bot"
)

func main() {
	config := irc.IRCBotConfig{
		Host: "irc.example.com",
		Port: "6697",
		TlsConfig: &tls.Config{
			// self accecpt certificate (can be done better)
			InsecureSkipVerify: true,
		},
		Password: "rng passowrd",
		Nickname: "your_bot_nick",
		Username: "your_username",
		QuitMessage: "Goodbye Cruel World",
	}

	bot := irc.NewIRCBot(config)

	// bot.AddBotCommand("!echo", func(bot *irc.IRCBot, input string) {
	// 	bot.SendMsg(fmt.Sprintf("PRIVMSG #test %s\r\n", input))
	// })

	bot.Connect()
	bot.Login()
	bot.Serve()
}
