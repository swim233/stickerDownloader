package hashCalculator

import (
	"crypto/md5"
	"encoding/hex"
	"regexp"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/utils"
)

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

// 计算哈希
func CalculateHash(u tgbotapi.Update) (hash string, err error) {
	setName := getStickerSet(u)
	stickerSet, err := utils.Bot.GetStickerSet(tgbotapi.GetStickerSetConfig{Name: setName})
	if err != nil {
		return "", err
	}
	var byteData []byte
	for _, v := range stickerSet.Stickers {
		byteData = append(byteData, []byte(v.FileID)...)
	}
	sum := md5.Sum(byteData)
	return hex.EncodeToString(sum[:]), nil
}

// 贴纸名计算哈希
func CalculateHashViaSetName(setName string) (hash string) {
	stickerSet, err := utils.Bot.GetStickerSet(tgbotapi.GetStickerSetConfig{Name: setName})
	if err != nil {
		return ""
	}
	var byteData []byte
	for _, v := range stickerSet.Stickers {
		byteData = append(byteData, []byte(v.FileID)...)
	}
	sum := md5.Sum(byteData)
	return hex.EncodeToString(sum[:])
}
