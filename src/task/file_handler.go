package task

import (
	"fmt"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/spf13/viper"
	"github.com/swim233/StickerDownloader/config"
	"github.com/swim233/StickerDownloader/core"
	"github.com/swim233/StickerDownloader/utils"
)

// 获取文件FileID
func getFileID(u tgbotapi.Update) string {
	fileID := u.CallbackQuery.Message.ReplyToMessage.Sticker.FileID
	return fileID
}

// 获取贴纸集url
func getSetUrl(sticker tgbotapi.Sticker) (url string, err error) {
	fileID := sticker.FileID
	FileURL, err := func(bot tgbotapi.BotAPI, fileID string) (string, error) {
		file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", bot.Token, file.FilePath), nil
	}(*core.Bot, fileID)
	if err != nil {
		utils.RuntimeStatus.Errors++
		return "", err
	}
	return FileURL, nil
}

// 获取文件url
func getUrl(u tgbotapi.Update) (url string, err error) {
	fileID := getFileID(u)
	var fileURL string
	var Err error
	for i := 0; i < viper.GetInt("max_retry"); i++ {

		file, err := core.Bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
		if err != nil {
			//FIXME:need log warning

			continue
		} else {
			fileURL = file.Link(config.BotToken)
			return fileURL, nil
		}
	}

	return "", Err
}
