package message

import (
	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/core"
	"github.com/swim233/StickerDownloader/db"
	"github.com/swim233/StickerDownloader/lib"
)

func GenerateNewTaskMessage(userID, chatID int64, ReplyMsgID int) {

	buttons := func() []tgbotapi.InlineKeyboardButton {
		var buttons []tgbotapi.InlineKeyboardButton
		button1 := tgbotapi.NewInlineKeyboardButtonData(lib.TranslationsMap[db.GetUserLanguage(userID)].DownloadSingleSticker, lib.SingleDownload.String())

		button2 := tgbotapi.NewInlineKeyboardButtonData(lib.TranslationsMap[db.GetUserLanguage(userID)].DownloadStickerPack, lib.SetDownload.String())

		buttons = append(buttons, button1, button2)
		return buttons
	}()
	cancelButton := tgbotapi.NewInlineKeyboardRow(generateCancelMessage(lib.TranslationsMap[db.GetUserLanguage(userID)].Cancel))

	msg := tgbotapi.NewMessage(chatID, lib.TranslationsMap[db.GetUserLanguage(chatID)].PickDownloadMethod)

	markup := tgbotapi.NewInlineKeyboardMarkup(buttons, cancelButton)

	msg.ReplyMarkup = markup
	msg.ReplyToMessageID = ReplyMsgID
	core.Bot.Send(msg)
}

func generateCancelMessage(text string) tgbotapi.InlineKeyboardButton {
	button := tgbotapi.NewInlineKeyboardButtonData(text, "cancel")
	return button
}
