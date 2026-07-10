package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/swim233/StickerDownloader/config"
	"github.com/swim233/StickerDownloader/handler"
	"github.com/swim233/StickerDownloader/logger"
)

// RunHTTPServer starts the HTTP API server and stops it when the context is canceled.
func RunHTTPServer(ctx context.Context) error {
	if !config.EnableHTTPServer {
		logger.Info("HTTP 服务器未开启")
		<-ctx.Done()
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/stickerpack", handleStickerPack)
	server := &http.Server{
		Addr:              config.HTTPServerPort,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), config.SupervisorShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Warn("HTTP 服务器关闭失败: %s", err)
		}
	}()

	logger.Info("HTTP 服务器已开启，端口: %s", config.HTTPServerPort)
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) && ctx.Err() != nil {
		<-shutdownDone
		return nil
	}
	return err
}

// StartHTTPServer preserves the old entry point for external callers.
func StartHTTPServer() {
	if err := RunHTTPServer(context.Background()); err != nil {
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
		_ = json.NewEncoder(w).Encode(stickerData)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename="+name+".zip")
	_, _ = w.Write(stickerData)
}
