package task

import (
	"fmt"
	"regexp"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/config"
	"github.com/swim233/StickerDownloader/core"
	"github.com/swim233/StickerDownloader/logger"
)

// getFileID extracts the sticker FileID from a callback query.
func getFileID(u tgbotapi.Update) string {
	return u.CallbackQuery.Message.ReplyToMessage.Sticker.FileID
}

// getUrl resolves a Telegram file ID to a download URL with retry.
func getUrl(fileID string) (string, error) {
	var lastErr error
	for i := 0; i < config.MaxRetry; i++ {
		file, err := core.Bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
		if err != nil {
			logger.Warn("获取文件 URL 失败 (重试 %d/%d): %s", i+1, config.MaxRetry, err)
			lastErr = err
			continue
		}
		logger.Debug("已解析 Telegram 文件路径")
		return fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", config.BotToken, file.FilePath), nil
	}
	return "", lastErr
}

// getSetUrl resolves a sticker's file URL.
func getSetUrl(sticker tgbotapi.Sticker) (string, error) {

	return getUrl(sticker.FileID)
}

// getStickerSetName extracts the sticker set name from a callback query.
func getStickerSetName(u tgbotapi.Update) string {

	var stickerLinkRegex = regexp.MustCompile(`https://t.me/addstickers/([a-zA-Z0-9_]+)`)

	if u.CallbackQuery != nil && u.CallbackQuery.Message.ReplyToMessage.Sticker != nil {
		return u.CallbackQuery.Message.ReplyToMessage.Sticker.SetName
	}
	if u.CallbackQuery != nil && u.CallbackQuery.Message.ReplyToMessage.Text != "" {
		matches := stickerLinkRegex.FindStringSubmatch(u.CallbackQuery.Message.ReplyToMessage.Text)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}
