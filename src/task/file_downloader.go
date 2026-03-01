package task

import (
	"errors"
	"io"
	"net/http"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/core"
	"github.com/swim233/StickerDownloader/db"
	"github.com/swim233/StickerDownloader/lib"
	"github.com/swim233/StickerDownloader/logger"
)

type FileDownloader struct {
}

func (f *FileDownloader) DownloadSingleSticker(update tgbotapi.Update) (data []byte, err error) {
	if update.Message == nil {
		logger.Warn("message is nil")
		return nil, errors.New("message is nil")
	}
	if update.CallbackQuery.Message.ReplyToMessage == nil {
		msg := tgbotapi.DeleteMessageConfig{
			ChatID:    update.Message.Chat.ID,
			MessageID: update.Message.MessageID,
		}
		_, err := core.Bot.Request(msg)
		if err != nil {
			logger.Warn("error in deleting msg: %s", err.Error())
		}
	}
	if update.CallbackQuery.Message.Sticker != nil {
		//TODO:检查缓存
		url, err := getUrl(getFileID(update))
		if err != nil {
			sendDownloadErrMsg("error in getting fileID", update, err)
		}
		rsps, err := http.Get(url)
		defer rsps.Body.Close()
		if err != nil {
			sendDownloadErrMsg("error in downloading file", update, err)
		}
		data, err = io.ReadAll(rsps.Body)
		if err != nil {
			sendDownloadErrMsg("error in downloading file", update, err)
		}
		return data, nil
	}
	return nil, errors.New("can not found sticker")
}

func (f *FileDownloader) DownloadStickerSet(update tgbotapi.Update) (data []byte, err error) {
	panic("not implemented") // TODO: Implement
}

func sendDownloadErrMsg(msg string, update tgbotapi.Update, err error) (any, error) {
	logger.Warn(msg+": %s", err.Error())
	errMsg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, lib.TranslationsMap[db.GetUserLanguage(update.CallbackQuery.From.ID)].FailToDownload)
	core.Bot.Send(errMsg)
	return nil, err
}
