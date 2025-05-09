package hashCalculator

import (
	"crypto/md5"
	"encoding/hex"
	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
)

func CalculateStickerSet(stickerSet tgbotapi.StickerSet) (hash string) {
	var byteData []byte
	for _, v := range stickerSet.Stickers {
		byteData = append(byteData, []byte(v.FileID)...) // 业务需要FileId唯一性以保证数据库缓存可以被使用 别换成FileUniqueId
	}
	sum := md5.Sum(byteData)
	return hex.EncodeToString(sum[:])
}
