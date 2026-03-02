package message

import (
	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/stickerDownloader/core"
)

func SendDownloadMethodSelectMessage(u tgbotapi.Update) {

	msg := tgbotapi.NewMessage(u.Message.Chat.ID, "select download method")

	msg.ReplyToMessageID = u.Message.MessageID

	msg.ReplyMarkup = makeButtonMarkUp()

	core.Bot.Send(msg)
}

func makeButtonMarkUp() tgbotapi.InlineKeyboardMarkup {
	singleButton := tgbotapi.NewInlineKeyboardButtonData("single", "single")
	setButton := tgbotapi.NewInlineKeyboardButtonData("set", "set")

	funcButtons := append([]tgbotapi.InlineKeyboardButton{}, singleButton, setButton)

	markup := tgbotapi.NewInlineKeyboardMarkup(funcButtons)
	return markup
}
