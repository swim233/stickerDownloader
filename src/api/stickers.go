package api

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"sync"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/core"
	"github.com/swim233/StickerDownloader/handler"
	logger "github.com/swim233/StickerDownloader/logger"
)

// stickerCacheMax bounds the in-memory sticker media cache (FIFO eviction).
const stickerCacheMax = 256

var stickerCache = struct {
	mu    sync.Mutex
	data  map[string][]byte
	order []string
}{data: map[string][]byte{}}

func stickerCacheGet(fileID string) ([]byte, bool) {
	stickerCache.mu.Lock()
	defer stickerCache.mu.Unlock()
	data, ok := stickerCache.data[fileID]
	return data, ok
}

func stickerCachePut(fileID string, data []byte) {
	stickerCache.mu.Lock()
	defer stickerCache.mu.Unlock()
	if _, ok := stickerCache.data[fileID]; ok {
		return
	}
	if len(stickerCache.order) >= stickerCacheMax {
		oldest := stickerCache.order[0]
		stickerCache.order = stickerCache.order[1:]
		delete(stickerCache.data, oldest)
	}
	stickerCache.data[fileID] = data
	stickerCache.order = append(stickerCache.order, fileID)
}

// stickerContentType sniffs the media type of downloaded sticker bytes.
func stickerContentType(data []byte) string {
	switch {
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return "image/webp"
	case bytes.HasPrefix(data, []byte{0x1A, 0x45, 0xDF, 0xA3}):
		return "video/webm"
	case bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G'}):
		return "image/png"
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return "image/jpeg"
	case bytes.HasPrefix(data, []byte{0x1F, 0x8B}):
		// Animated .tgs sticker: gzipped Lottie JSON, unwrapped by handleAPISticker.
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}

// handleAPISticker proxies a sticker file so the WebUI can display it.
func handleAPISticker(w http.ResponseWriter, r *http.Request) {
	fileID := r.URL.Query().Get("file_id")
	if fileID == "" {
		writeJSONError(w, http.StatusBadRequest, "缺少 file_id 参数")
		return
	}

	data, ok := stickerCacheGet(fileID)
	if !ok {
		var err error
		data, err = handler.StickerDownloader{}.DownloadFile(fileID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "获取贴纸文件失败")
			return
		}
		stickerCachePut(fileID, data)
	}

	// Telegram file IDs are immutable, so long client-side caching is safe.
	w.Header().Set("Cache-Control", "private, max-age=604800")

	// A .tgs sticker is gzipped Lottie JSON; unwrap it so the WebUI's player
	// can consume it directly.
	if contentType := stickerContentType(data); contentType == "application/gzip" {
		lottie, err := unwrapTGS(data)
		if err != nil {
			logger.Warn("解压 TGS 贴纸失败: %s", err)
			writeJSONError(w, http.StatusUnsupportedMediaType, "无法解析动画贴纸")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(lottie)
		return
	} else {
		w.Header().Set("Content-Type", contentType)
	}
	_, _ = w.Write(data)
}

// maxLottieBytes bounds decompression so a malformed .tgs can't exhaust memory.
const maxLottieBytes = 8 << 20

// unwrapTGS gunzips a .tgs sticker into its Lottie JSON payload.
func unwrapTGS(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	lottie, err := io.ReadAll(io.LimitReader(reader, maxLottieBytes+1))
	if err != nil {
		return nil, err
	}
	if len(lottie) > maxLottieBytes {
		return nil, errors.New("动画数据过大")
	}
	return lottie, nil
}

type stickerSetItem struct {
	FileID     string `json:"file_id"`
	Emoji      string `json:"emoji,omitempty"`
	IsVideo    bool   `json:"is_video,omitempty"`
	IsAnimated bool   `json:"is_animated,omitempty"`
}

type stickerSetResponse struct {
	Name     string           `json:"name"`
	Title    string           `json:"title"`
	Stickers []stickerSetItem `json:"stickers"`
}

// handleAPIStickerSet lists all stickers in a set for the WebUI detail view.
func handleAPIStickerSet(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "缺少 name 参数")
		return
	}
	set, err := core.Bot.GetStickerSet(tgbotapi.GetStickerSetConfig{Name: name})
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "获取贴纸包失败")
		return
	}
	resp := stickerSetResponse{Name: set.Name, Title: set.Title, Stickers: make([]stickerSetItem, 0, len(set.Stickers))}
	for _, sticker := range set.Stickers {
		resp.Stickers = append(resp.Stickers, stickerSetItem{
			FileID:     sticker.FileID,
			Emoji:      sticker.Emoji,
			IsVideo:    sticker.IsVideo,
			IsAnimated: sticker.IsAnimated,
		})
	}
	writeJSON(w, resp)
}
