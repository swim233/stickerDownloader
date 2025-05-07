package cache

import (
	"errors"

	"github.com/swim233/StickerDownloader/utils/db"
	"github.com/swim233/StickerDownloader/utils/hashCalculator"
)

func GetCacheFileID(setName string, format string) (fileID string, fileSize int64, stickerNum int, err error) {
	stickerData, err := db.GetStickerData(setName)
	if err != nil {
		return "", 0, 0, err
	}
	if format == "webp" && hashCalculator.CalculateHashViaSetName(setName) == stickerData.SetHash {
		return stickerData.WebpFileID, stickerData.WebpFileSize, stickerNum, nil
	}
	if format == "png" && hashCalculator.CalculateHashViaSetName(setName) == stickerData.SetHash {
		return stickerData.PNGFileID, stickerData.PNGFileSize, stickerNum, nil
	}
	if format == "jpeg" && hashCalculator.CalculateHashViaSetName(setName) == stickerData.SetHash {
		return stickerData.JPEGFileID, stickerData.JPEGFileSize, stickerNum, nil
	}
	return "", 0, 0, errors.New("哈希不匹配或发生其他错误")
}
