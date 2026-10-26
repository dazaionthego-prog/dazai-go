package main

import (
	"strings"

	tele "gopkg.in/telebot.v3"
)

type buyPlan struct {
	Name    string
	EmojiID string
	Emoji   string
	Expiry  string
	Price   string
}

var buyPlans = []buyPlan{
	{Name: "Gold", EmojiID: "5440539497383087970", Emoji: "🥇", Expiry: "15 days", Price: "$8.00"},
	{Name: "Silver", EmojiID: "5447203607294265305", Emoji: "🥈", Expiry: "8 days", Price: "$4.00"},
	{Name: "Bronze", EmojiID: "5453902265922376865", Emoji: "🥉", Expiry: "4 days", Price: "$2.00"},
}

func buyPlanLine(p buyPlan) string {
	return "<b>Plan name:</b> <tg-emoji emoji-id=\"" + p.EmojiID + "\">" + p.Emoji + "</tg-emoji> " +
		"<b>" + p.Name + "</b>\n" +
		"<b>Plan expiry:</b> <b>" + p.Expiry + "</b>\n" +
		"<b>Plan price:</b> <b>" + p.Price + "</b>\n" +
		"━━━━━━━━━━━━━━━━━━"
}

var buyHTML string

func init() {
	parts := make([]string, 0, len(buyPlans))
	for _, p := range buyPlans {
		parts = append(parts, buyPlanLine(p))
	}
	buyHTML = "<tg-emoji emoji-id=\"5395463407589672312\">🟧</tg-emoji>" +
		"<b>Welcome to Dazai's store:</b>\n\n" +
		"<b>Here you can buy and get info about bot's plan.</b>\n" +
		"━━━━━━━━━━━━━━━━━━\n" +
		strings.Join(parts, "\n") + "\n" +
		"<b>To buy any of these plans, please drop a message at: " +
		"@dazaihelpbot</b>\n" +
		"<b>Thank you.</b>"
}

func SetupBuy(bot *tele.Bot) {
	bot.Handle("/buy", func(c tele.Context) error {
		m := c.Message()
		if m == nil {
			return nil
		}
		_reply(bot, m, buyHTML)
		return nil
	})
}
