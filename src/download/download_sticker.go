package download

import (
	"github.com/swim233/stickerDownloader/lib"
	"github.com/swim233/stickerDownloader/logger"
)

var DownloadRequestMap = make(map[lib.DownloadRequestID]*lib.DownloadRequest)

func OperateDownloadRequest(chatID int64, messageID int, req *lib.DownloadRequest) {

	logger.Debug("orating download request: %v", req)
	DownloadRequestMap[lib.GetRequestID(chatID, messageID)] = req

}

func EditDownloadRequestType(chatID int64, messageID int, downloadType string) {
	logger.Debug("chatID: %d, massageID: %d, download type: %s", chatID, messageID, downloadType)
	DownloadRequestMap[lib.GetRequestID(chatID, messageID)].DownloadType = downloadType
}
