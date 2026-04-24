package api

import (
	"encoding/json"
	"net/http"

	"github.com/swim233/StickerDownloader/config"
	"github.com/swim233/StickerDownloader/handler"
	"github.com/swim233/StickerDownloader/logger"
)

// StartHTTPServer starts the HTTP API server if enabled in config.
func StartHTTPServer() {
	if !config.EnableHTTPServer {
		logger.Info("HTTP 服务器未开启")
		return
	}

	logger.Info("HTTP 服务器已开启，端口: %s", config.HTTPServerPort)
	http.HandleFunc("/stickerpack", handleStickerPack)

	if err := http.ListenAndServe(config.HTTPServerPort, nil); err != nil {
		logger.Error("HTTP 服务器错误: %s", err)
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
		format = "webp"
	}

	download := r.URL.Query().Get("download") == "true"
	dl := handler.StickerDownloader{}

	stickerData, err := dl.HTTPDownloadStickerSet(format, name)
	if err != nil {
		http.Error(w, "Failed to get sticker set: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !download {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stickerData)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename="+name+".zip")
	w.Write(stickerData)
}
