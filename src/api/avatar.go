package api

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/core"
	"github.com/swim233/StickerDownloader/handler"
)

// avatarTTL bounds how long a resolved (or missing) avatar is reused before
// Telegram is asked again.
const avatarTTL = 30 * time.Minute

type avatarEntry struct {
	data     []byte
	resolved time.Time
}

var avatarCache = struct {
	mu      sync.Mutex
	entries map[int64]avatarEntry
}{entries: map[int64]avatarEntry{}}

func avatarCacheGet(userID int64) ([]byte, bool) {
	avatarCache.mu.Lock()
	defer avatarCache.mu.Unlock()
	entry, ok := avatarCache.entries[userID]
	if !ok || time.Since(entry.resolved) > avatarTTL {
		return nil, false
	}
	return entry.data, true
}

func avatarCachePut(userID int64, data []byte) {
	avatarCache.mu.Lock()
	defer avatarCache.mu.Unlock()
	avatarCache.entries[userID] = avatarEntry{data: data, resolved: time.Now()}
}

// fetchAvatar returns the user's smallest profile photo, or nil when the user
// has none (or hides them).
func fetchAvatar(userID int64) []byte {
	photos, err := core.Bot.GetUserProfilePhotos(tgbotapi.UserProfilePhotosConfig{UserID: userID, Limit: 1})
	if err != nil || len(photos.Photos) == 0 || len(photos.Photos[0]) == 0 {
		return nil
	}
	// Sizes come smallest-first; the first one is enough for a 40px avatar.
	data, err := handler.StickerDownloader{}.DownloadFile(photos.Photos[0][0].FileID)
	if err != nil {
		return nil
	}
	return data
}

// handleAPIAvatar proxies a user's Telegram profile photo. A 404 tells the
// WebUI to fall back to its generated initial avatar.
func handleAPIAvatar(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(r.URL.Query().Get("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "user_id 参数无效")
		return
	}

	data, cached := avatarCacheGet(userID)
	if !cached {
		data = fetchAvatar(userID)
		avatarCachePut(userID, data)
	}
	if len(data) == 0 {
		writeJSONError(w, http.StatusNotFound, "该用户没有头像")
		return
	}

	w.Header().Set("Cache-Control", "private, max-age=1800")
	w.Header().Set("Content-Type", stickerContentType(data))
	_, _ = w.Write(data)
}
