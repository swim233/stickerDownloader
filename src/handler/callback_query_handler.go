package handler

import (
	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/stickerDownloader/download"
	"github.com/swim233/stickerDownloader/logger"
	"github.com/swim233/stickerDownloader/message"
)

func AddSingleCallbackQueryHandler(b *tgbotapi.Bot) {

	b.NewCallBackProcessor("single", func(update tgbotapi.Update) error {

		logger.Debug("get single download callback")
		if isCallbackWithSticker(update) {

			logger.Debug("get single callback")
			download.EditDownloadRequestType(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, "single")
			message.SendDownloadStartMessage(update)
			
		}
		return nil
	})
}

func isCallbackWithSticker(update tgbotapi.Update) bool {
	return update.CallbackQuery.Message.ReplyToMessage != nil && update.CallbackQuery.Message.ReplyToMessage.Sticker != nil
}
