package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandleAPIAvatarInvalidID(t *testing.T) {
	rec := httptest.NewRecorder()
	handleAPIAvatar(rec, httptest.NewRequest(http.MethodGet, "/api/avatar?user_id=abc", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleAPIAvatarServesCachedPhoto(t *testing.T) {
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 'x'}
	avatarCachePut(555, jpeg)
	t.Cleanup(func() {
		avatarCache.mu.Lock()
		delete(avatarCache.entries, 555)
		avatarCache.mu.Unlock()
	})

	rec := httptest.NewRecorder()
	handleAPIAvatar(rec, httptest.NewRequest(http.MethodGet, "/api/avatar?user_id=555", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Fatalf("Content-Type = %s, want image/jpeg", ct)
	}
	if rec.Body.String() != string(jpeg) {
		t.Fatal("body mismatch")
	}
}

func TestAvatarCacheRemembersMissingPhoto(t *testing.T) {
	avatarCachePut(556, nil)
	t.Cleanup(func() {
		avatarCache.mu.Lock()
		delete(avatarCache.entries, 556)
		avatarCache.mu.Unlock()
	})

	// A negative result is still a cache hit, so Telegram isn't re-queried.
	data, cached := avatarCacheGet(556)
	if !cached || data != nil {
		t.Fatalf("cache get = %v, %v; want cached nil", data, cached)
	}

	rec := httptest.NewRecorder()
	handleAPIAvatar(rec, httptest.NewRequest(http.MethodGet, "/api/avatar?user_id=556", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestAvatarCacheExpires(t *testing.T) {
	avatarCache.mu.Lock()
	avatarCache.entries[557] = avatarEntry{data: []byte("x"), resolved: time.Now().Add(-2 * avatarTTL)}
	avatarCache.mu.Unlock()
	t.Cleanup(func() {
		avatarCache.mu.Lock()
		delete(avatarCache.entries, 557)
		avatarCache.mu.Unlock()
	})

	if _, cached := avatarCacheGet(557); cached {
		t.Fatal("stale entry should miss")
	}
}
