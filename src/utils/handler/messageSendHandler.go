package handler

import (
	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	utils "github.com/swim233/StickerDownloader/utils"
	"github.com/swim233/StickerDownloader/utils/logger"
)

type MessageSender struct {
}

func (m MessageSender) MessageSender(u tgbotapi.Update) error {
	dl := StickerDownloader{}
	chatID := u.Message.From.ID
	data, stickerSetName, _ := dl.DownloadStickerSet(u)
	msg := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{Name: stickerSetName + ".zip", Bytes: data})
	_, err := utils.Bot.Send(msg)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	return nil
}

func (m MessageSender) ButtonMessageSender(u tgbotapi.Update) error {
	chatID := u.Message.From.ID
	msg := tgbotapi.NewMessage(chatID, "请选择要下载的方式")
	msg.ReplyToMessageID = u.Message.MessageID
	button1 := tgbotapi.NewInlineKeyboardButtonData("下载单个图片", "this")
	button2 := tgbotapi.NewInlineKeyboardButtonData("下载贴纸包", "zip")
	msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{button1}, {button2}}}
	utils.Bot.Send(msg)
	return nil
}
func (m MessageSender) ThisSender(u tgbotapi.Update) error {
	chatID := u.CallbackQuery.Message.Chat.ID
	dl := StickerDownloader{}
	msg := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{Bytes: func(u tgbotapi.Update) []byte {
		data, _ := dl.DownloadFile(u)
		return data
	}(u)})
	utils.Bot.Send(msg)
	u.CallbackQuery.Answer(false, "正在下载单个图片")
	u.CallbackQuery.Delete()
	return nil
}

func (m MessageSender) ZipSender(u tgbotapi.Update) error {
	chatID := u.CallbackQuery.Message.Chat.ID
	dl := StickerDownloader{}
	data, stickerSetName, _ := dl.DownloadStickerSet(u)
	msg := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{Name: stickerSetName + ".zip", Bytes: data})
	utils.Bot.Send(msg)
	u.CallbackQuery.Answer(false, "正在下载贴纸包")
	u.CallbackQuery.Delete()
	return nil
}
