package message

import (
	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/core"
)

func GenerateNewTaskMessage(chatID int64, ReplyMsgID int) {
	buttons := func() []tgbotapi.InlineKeyboardButton {
		var buttons []tgbotapi.InlineKeyboardButton
		button1 := tgbotapi.NewInlineKeyboardButtonData("button1", "button1")
		button2 := tgbotapi.NewInlineKeyboardButtonData("button2", "button2")
		buttons = append(buttons, button1, button2)
		return buttons
	}()
	tgbotapi.NewInlineKeyboardRow(generateCancelMessage("cancel"))
	msg := tgbotapi.NewMessage(chatID, "test msg")
	markup := tgbotapi.NewInlineKeyboardMarkup(buttons)
	msg.ReplyMarkup = markup
	msg.ReplyToMessageID = ReplyMsgID
	core.Bot.Send(msg)
}

func generateCancelMessage(text string) tgbotapi.InlineKeyboardButton {
	button := tgbotapi.NewInlineKeyboardButtonData(text, "cancel")
	return button
}
