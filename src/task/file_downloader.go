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

// FileDownloader implements the Downloader interface for Telegram sticker files.
type FileDownloader struct{}

func (f *FileDownloader) DownloadSingleSticker(update tgbotapi.Update) ([]byte, error) {
	// Find the sticker from either Message or CallbackQuery
	var sticker *tgbotapi.Sticker
	var chatID int64

	if update.CallbackQuery != nil &&
		update.CallbackQuery.Message != nil &&
		update.CallbackQuery.Message.ReplyToMessage != nil &&
		update.CallbackQuery.Message.ReplyToMessage.Sticker != nil {
		sticker = update.CallbackQuery.Message.ReplyToMessage.Sticker
		chatID = update.CallbackQuery.Message.Chat.ID
	} else if update.Message != nil && update.Message.Sticker != nil {
		sticker = update.Message.Sticker
		chatID = update.Message.Chat.ID
	}

	if sticker == nil {
		return nil, errors.New("sticker not found")
	}

	url, err := getUrl(sticker.FileID)
	if err != nil {
		sendDownloadErrMsg("获取文件 URL 出错", update, chatID, err)
		return nil, err
	}

	resp, err := http.Get(url)
	if err != nil {
		sendDownloadErrMsg("下载文件出错", update, chatID, err)
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		sendDownloadErrMsg("读取文件出错", update, chatID, err)
		return nil, err
	}
	return data, nil
}

func (f *FileDownloader) DownloadStickerSet(update tgbotapi.Update) ([]byte, error) {
	// TODO: Implement sticker set download via task system
	return nil, errors.New("not implemented")
}

func sendDownloadErrMsg(msg string, update tgbotapi.Update, chatID int64, err error) {
	logger.Warn("%s: %s", msg, err)
	var userID int64
	if update.CallbackQuery != nil && update.CallbackQuery.From != nil {
		userID = update.CallbackQuery.From.ID
	}
	if chatID != 0 && userID != 0 {
		errMsg := tgbotapi.NewMessage(
			chatID,
			lib.TranslationsMap[db.GetUserLanguage(userID)].FailToDownload,
		)
		core.Bot.Send(errMsg)
	}
}
