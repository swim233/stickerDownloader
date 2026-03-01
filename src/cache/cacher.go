package cache

import (
	"errors"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"

	"github.com/swim233/StickerDownloader/db"
	"github.com/swim233/StickerDownloader/lib"
	"github.com/swim233/StickerDownloader/utils"
)

func GetCacheFileID(set tgbotapi.StickerSet, format lib.TaskFileFormat) (fileID string, fileSize int64, stickerNum int, err error) {
	stickerData, err := db.GetStickerData(set.Name)
	if err != nil {
		return "", 0, 0, err
	}
	if format == lib.WebpFormat && utils.CalculateStickerSet(set) == stickerData.SetHash {
		return stickerData.WebpFileID, stickerData.WebpFileSize, stickerNum, nil
	}
	if format == lib.PngFormat && utils.CalculateStickerSet(set) == stickerData.SetHash {
		return stickerData.PNGFileID, stickerData.PNGFileSize, stickerNum, nil
	}
	if format == lib.JpegFormat && utils.CalculateStickerSet(set) == stickerData.SetHash {
		return stickerData.JPEGFileID, stickerData.JPEGFileSize, stickerNum, nil
	}
	return "", 0, 0, errors.New("哈希不匹配或发生其他错误")
}
