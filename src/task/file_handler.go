package task

import (
	"regexp"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/spf13/viper"
	"github.com/swim233/StickerDownloader/config"
	"github.com/swim233/StickerDownloader/core"
	"github.com/swim233/StickerDownloader/logger"
)

// 获取文件FileID
func getFileID(u tgbotapi.Update) string {
	fileID := u.CallbackQuery.Message.ReplyToMessage.Sticker.FileID
	return fileID
}

// 获取贴纸集url
func getSetUrl(sticker tgbotapi.Sticker) (url string, err error) {
	return getUrl(sticker.FileID)

}

// 获取文件url
func getUrl(fileID string) (url string, err error) {
	var Err error
	for i := 0; i < viper.GetInt("max_retry"); i++ {

		file, err := core.Bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
		if err != nil {
			logger.Warn("error in getting file url: %s", err.Error())
			Err = err
			continue
		} else {
			url = file.Link(config.BotToken)
			return url, nil
		}
	}

	return "", Err
}

// 获取贴纸集
func getStickerSet(u tgbotapi.Update) string {
	var stickerLinkRegex = regexp.MustCompile(`https://t.me/addstickers/([a-zA-Z0-9_]+)`)

	if u.CallbackQuery != nil && u.CallbackQuery.Message.ReplyToMessage.Sticker != nil {
		return u.CallbackQuery.Message.ReplyToMessage.Sticker.SetName
	}

	if u.CallbackQuery.Message.ReplyToMessage.Text != "" && stickerLinkRegex.MatchString(u.CallbackQuery.Message.ReplyToMessage.Text) {
		// 提取 sticker set name
		matches := stickerLinkRegex.FindStringSubmatch(u.CallbackQuery.Message.ReplyToMessage.Text)
		if len(matches) > 1 {
			stickerSetName := matches[1] // 提取的 SetName
			return stickerSetName
		}
	}
	return ""
}
