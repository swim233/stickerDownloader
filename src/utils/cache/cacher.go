package cache

import (
	"errors"
	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/utils"

	"github.com/swim233/StickerDownloader/utils/db"
	"github.com/swim233/StickerDownloader/utils/hashCalculator"
)

func GetCacheFileID(set tgbotapi.StickerSet, format utils.Format) (fileID string, fileSize int64, stickerNum int, err error) {
	stickerData, err := db.GetStickerData(set.Name)
	if err != nil {
		return "", 0, 0, err
	}
	if format == utils.WebpFormat && hashCalculator.CalculateStickerSet(set) == stickerData.SetHash {
		return stickerData.WebpFileID, stickerData.WebpFileSize, stickerNum, nil
	}
	if format == utils.PngFormat && hashCalculator.CalculateStickerSet(set) == stickerData.SetHash {
		return stickerData.PNGFileID, stickerData.PNGFileSize, stickerNum, nil
	}
	if format == utils.JpegFormat && hashCalculator.CalculateStickerSet(set) == stickerData.SetHash {
		return stickerData.JPEGFileID, stickerData.JPEGFileSize, stickerNum, nil
	}
	return "", 0, 0, errors.New("哈希不匹配或发生其他错误")
}
