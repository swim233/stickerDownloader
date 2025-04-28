package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/swim233/StickerDownloader/utils"
	"github.com/swim233/StickerDownloader/utils/handler"
	"github.com/swim233/StickerDownloader/utils/logger"
)

func StartHTTPServer() {
	if utils.BotConfig.EnableHTTPServer {
		logger.Info("HTTP服务器已开启")
		http.HandleFunc("/stickerpack", handleStickerPack)

		port := utils.BotConfig.HTTPServerPort

		logger.Info("[HTTP] Server started on %s", port)
		err := http.ListenAndServe(port, nil)
		if err != nil {
			logger.Error("%s", err.Error())
		}
	} else {
		logger.Info("HTTP服务器未开启")
	}
}

func handleStickerPack(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Missing 'name' parameter", http.StatusBadRequest)
		return
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "webp" // 默认值
	}

	download := r.URL.Query().Get("download") == "true"
	hd := handler.StickerDownloader{}

	stickerSet, err := hd.HTTPDownloadStickerSet(format, name)
	if err != nil {
		http.Error(w, "Failed to get sticker set: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !download {
		// 返回 JSON 信息
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stickerSet)
		return
	}

	// 设置返回头并输出 zip 文件
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename="+name+".zip")
	w.Write(stickerSet)
}
