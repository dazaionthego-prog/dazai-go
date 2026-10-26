package main

import (
	"strconv"

	tele "gopkg.in/telebot.v3"
)

func SetupID(bot *tele.Bot) {
	bot.Handle("/id", func(c tele.Context) error {
		m := c.Message()
		if m == nil {
			return nil
		}

		reply := m.ReplyTo
		var user *tele.User
		if reply != nil && reply.Sender != nil && reply.SenderChat == nil {
			user = reply.Sender
		} else {
			user = m.Sender
		}
		if user == nil {
			return nil
		}

		text := "🆔<b>User ID!</b>\n" +
			"━━━━━━━━━━━━━━━━━━\n" +
			"<b>User:</b> " + _mention(user.ID, user.FirstName) + "\n" +
			"<b>UserID:</b> <code>" + strconv.FormatInt(user.ID, 10) + "</code>\n"

		if m.Chat != nil && m.Chat.Type != tele.ChatPrivate {
			text += "<b>ChatID:</b> <code>" + strconv.FormatInt(m.Chat.ID, 10) + "</code>\n"
		}
		text += "━━━━━━━━━━━━━━━━━━"

		_reply(bot, m, text)
		return nil
	})
}
