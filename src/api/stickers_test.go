package api

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStickerContentType(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"webp", []byte("RIFF\x00\x00\x00\x00WEBPVP8 "), "image/webp"},
		{"webm", []byte{0x1A, 0x45, 0xDF, 0xA3, 0x00}, "video/webm"},
		{"png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A}, "image/png"},
		{"tgs", []byte{0x1F, 0x8B, 0x08}, "application/gzip"},
		{"unknown", []byte("hello"), "application/octet-stream"},
	}
	for _, tc := range cases {
		if got := stickerContentType(tc.data); got != tc.want {
			t.Fatalf("%s: got %s, want %s", tc.name, got, tc.want)
		}
	}
}

func TestStickerCacheEvictsOldest(t *testing.T) {
	stickerCache.mu.Lock()
	stickerCache.data = map[string][]byte{}
	stickerCache.order = nil
	stickerCache.mu.Unlock()

	for i := range stickerCacheMax + 1 {
		stickerCachePut(fmt.Sprintf("file%d", i), []byte{byte(i)})
	}
	if _, ok := stickerCacheGet("file0"); ok {
		t.Fatal("oldest entry should have been evicted")
	}
	if _, ok := stickerCacheGet(fmt.Sprintf("file%d", stickerCacheMax)); !ok {
		t.Fatal("newest entry missing")
	}
	stickerCache.mu.Lock()
	if len(stickerCache.data) != stickerCacheMax || len(stickerCache.order) != stickerCacheMax {
		t.Fatalf("cache size = %d/%d, want %d", len(stickerCache.data), len(stickerCache.order), stickerCacheMax)
	}
	stickerCache.mu.Unlock()
}

func TestHandleAPIStickerRequiresFileID(t *testing.T) {
	rec := httptest.NewRecorder()
	handleAPISticker(rec, httptest.NewRequest(http.MethodGet, "/api/sticker", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleAPIStickerServesFromCache(t *testing.T) {
	webp := []byte("RIFF\x00\x00\x00\x00WEBPVP8 data")
	stickerCachePut("cached-file", webp)

	rec := httptest.NewRecorder()
	handleAPISticker(rec, httptest.NewRequest(http.MethodGet, "/api/sticker?file_id=cached-file", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/webp" {
		t.Fatalf("Content-Type = %s, want image/webp", ct)
	}
	if rec.Body.String() != string(webp) {
		t.Fatal("body mismatch")
	}
}

func TestHandleAPIBansReturnsJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	handleAPIBans(rec, httptest.NewRequest(http.MethodGet, "/api/bans", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp bansResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func TestUnwrapTGS(t *testing.T) {
	lottie := []byte(`{"v":"5.5.7","fr":60,"layers":[]}`)
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(lottie); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	tgs := buf.Bytes()

	if got := stickerContentType(tgs); got != "application/gzip" {
		t.Fatalf("content type = %s, want application/gzip", got)
	}
	got, err := unwrapTGS(tgs)
	if err != nil {
		t.Fatalf("unwrap: %v", err)
	}
	if string(got) != string(lottie) {
		t.Fatalf("unwrapped = %s, want %s", got, lottie)
	}

	if _, err := unwrapTGS([]byte("not gzip")); err == nil {
		t.Fatal("expected error for non-gzip input")
	}
}

func TestHandleAPIStickerServesTGSAsJSON(t *testing.T) {
	lottie := []byte(`{"v":"5.5.7","layers":[]}`)
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(lottie)
	_ = zw.Close()
	stickerCachePut("tgs-file", buf.Bytes())

	rec := httptest.NewRecorder()
	handleAPISticker(rec, httptest.NewRequest(http.MethodGet, "/api/sticker?file_id=tgs-file", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %s, want JSON", ct)
	}
	if rec.Body.String() != string(lottie) {
		t.Fatalf("body = %s, want the Lottie JSON", rec.Body.String())
	}
}

func TestStickerContentTypeJPEG(t *testing.T) {
	if got := stickerContentType([]byte{0xFF, 0xD8, 0xFF, 0xE0}); got != "image/jpeg" {
		t.Fatalf("got %s, want image/jpeg", got)
	}
}
