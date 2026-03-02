package handler

import tgbotapi "github.com/ijnkawakaze/telegram-bot-api"

func SingleCallbackQueryHandler(b *tgbotapi.Bot) {

	b.NewCallBackProcessor("single", func(update tgbotapi.Update) error {

		if isCallbackWithSticker(update) {
			

		}
		return nil
	})
}

func isCallbackWithSticker(update tgbotapi.Update) bool {
	return update.CallbackQuery.Message.ReplyToMessage != nil && update.CallbackQuery.Message.ReplyToMessage.Sticker != nil
}
