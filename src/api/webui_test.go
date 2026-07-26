package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/swim233/StickerDownloader/lib"
	"github.com/swim233/StickerDownloader/utils"
)

func TestHandleIndexServesWebUI(t *testing.T) {
	rec := httptest.NewRecorder()
	handleIndex(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(rec.Body.String(), "最近下载") {
		t.Fatal("index page missing history section")
	}
}

func TestHandleIndexRejectsOtherPaths(t *testing.T) {
	rec := httptest.NewRecorder()
	handleIndex(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestHandleAPIHistoryReturnsRecords(t *testing.T) {
	utils.DownloadHistory.Add(lib.DownloadRecord{
		Source:   lib.DownloadSourceBot,
		Kind:     lib.DownloadKindPack,
		UserName: "tester",
		SetName:  "test_set",
		Format:   "webp",
	})

	rec := httptest.NewRecorder()
	handleAPIHistory(rec, httptest.NewRequest(http.MethodGet, "/api/history", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp historyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Capacity <= 0 {
		t.Fatalf("capacity = %d, want > 0", resp.Capacity)
	}
	if len(resp.Records) == 0 || resp.Records[0].SetName != "test_set" {
		t.Fatalf("records = %+v, want newest record test_set first", resp.Records)
	}
}

func TestHandleAPIStatusReturnsJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	handleAPIStatus(rec, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var report utils.RuntimeStatusReport
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode: %v", err)
	}
}
