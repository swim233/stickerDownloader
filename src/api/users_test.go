package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/swim233/StickerDownloader/config"
	db "github.com/swim233/StickerDownloader/db"
	"gorm.io/gorm"
)

func setupUserTestDB(t *testing.T) {
	t.Helper()
	old := db.DB
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := database.AutoMigrate(&db.UserData{}, &db.BannedUserData{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.DB = database
	t.Cleanup(func() { db.DB = old })
}

func TestHandleAPIUserInvalidID(t *testing.T) {
	rec := httptest.NewRecorder()
	handleAPIUser(rec, httptest.NewRequest(http.MethodGet, "/api/user?user_id=abc", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleAPIUserReturnsStatsAndLeaderboard(t *testing.T) {
	setupUserTestDB(t)
	for _, user := range []db.UserData{
		{UserID: 1, FirstName: "Alpha", UserName: "alpha", DownloadFiles: 5, DownloadFileSize: 500},
		{UserID: 2, FirstName: "Beta", DownloadFiles: 50, DownloadFileSize: 9000},
	} {
		if err := db.DB.Create(&user).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	rec := httptest.NewRecorder()
	handleAPIUser(rec, httptest.NewRequest(http.MethodGet, "/api/user?user_id=1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var resp userResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.User.Found || resp.User.DownloadFiles != 5 || resp.User.Rank != 2 {
		t.Fatalf("user = %+v, want found with 5 files rank 2", resp.User)
	}
	if len(resp.Leaderboard) != 2 || resp.Leaderboard[0].UserID != 2 {
		t.Fatalf("leaderboard = %+v, want user 2 first", resp.Leaderboard)
	}

	rec = httptest.NewRecorder()
	handleAPIUser(rec, httptest.NewRequest(http.MethodGet, "/api/user?user_id=777", nil))
	var missing userResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &missing); err != nil {
		t.Fatalf("decode missing: %v", err)
	}
	if missing.User.Found || missing.User.UserID != 777 {
		t.Fatalf("missing user = %+v, want not found", missing.User)
	}
}

func banRequest(t *testing.T, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	switch path {
	case "/api/user/ban":
		handleAPIUserBan(rec, req)
	case "/api/user/unban":
		handleAPIUserUnban(rec, req)
	default:
		t.Fatalf("unknown path %s", path)
	}
	return rec
}

func TestHandleAPIUserBanAndUnban(t *testing.T) {
	setupUserTestDB(t)

	if rec := banRequest(t, "/api/user/ban", `{"user_id":42,"reason":"abuse","silent":true}`); rec.Code != http.StatusOK {
		t.Fatalf("ban status = %d: %s", rec.Code, rec.Body.String())
	}
	ban, banned := db.GetBan(42)
	if !banned || !ban.Silent || ban.Reason != "abuse" {
		t.Fatalf("GetBan(42) = %+v, %v", ban, banned)
	}

	if rec := banRequest(t, "/api/user/unban", `{"user_id":42}`); rec.Code != http.StatusOK {
		t.Fatalf("unban status = %d: %s", rec.Code, rec.Body.String())
	}
	if _, banned := db.GetBan(42); banned {
		t.Fatal("user 42 still banned after unban")
	}
}

func TestHandleAPIUserBanRejectsOwner(t *testing.T) {
	setupUserTestDB(t)
	oldOwner := config.OwnerChatID
	config.OwnerChatID = 99
	t.Cleanup(func() { config.OwnerChatID = oldOwner })

	if rec := banRequest(t, "/api/user/ban", `{"user_id":99}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("ban owner status = %d, want 400", rec.Code)
	}
}

func TestHandleAPIUserBanRejectsGet(t *testing.T) {
	rec := httptest.NewRecorder()
	handleAPIUserBan(rec, httptest.NewRequest(http.MethodGet, "/api/user/ban", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestHandleAPILeaderboard(t *testing.T) {
	setupUserTestDB(t)
	for _, user := range []db.UserData{
		{UserID: 1, FirstName: "Alpha", DownloadFiles: 5, DownloadFileSize: 500},
		{UserID: 2, FirstName: "Beta", DownloadFiles: 50, DownloadFileSize: 9000},
		{UserID: 3, FirstName: "Gamma", DownloadFiles: 20, DownloadFileSize: 2000},
	} {
		if err := db.DB.Create(&user).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	rec := httptest.NewRecorder()
	handleAPILeaderboard(rec, httptest.NewRequest(http.MethodGet, "/api/leaderboard", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	var resp leaderboardResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Users) != 3 || resp.Users[0].UserID != 2 || resp.Users[2].UserID != 1 {
		t.Fatalf("ranking wrong: %+v", resp.Users)
	}
	if resp.Total != 3 {
		t.Fatalf("total = %d, want 3", resp.Total)
	}

	// limit is honoured
	rec = httptest.NewRecorder()
	handleAPILeaderboard(rec, httptest.NewRequest(http.MethodGet, "/api/leaderboard?limit=2", nil))
	var limited leaderboardResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &limited)
	if len(limited.Users) != 2 {
		t.Fatalf("limit ignored: %d users", len(limited.Users))
	}

	// and validated
	rec = httptest.NewRecorder()
	handleAPILeaderboard(rec, httptest.NewRequest(http.MethodGet, "/api/leaderboard?limit=abc", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad limit status = %d, want 400", rec.Code)
	}
}

func TestHandleAPILeaderboardSortAndSearch(t *testing.T) {
	setupUserTestDB(t)
	for _, user := range []db.UserData{
		{UserID: 1, FirstName: "Alpha", UserName: "alphawolf", DownloadFiles: 100, DownloadFileSize: 1000},
		{UserID: 2, FirstName: "Beta", UserName: "betauser", DownloadFiles: 10, DownloadFileSize: 999999},
		{UserID: 3, FirstName: "小", LastName: "明", DownloadFiles: 50, DownloadFileSize: 5000},
	} {
		if err := db.DB.Create(&user).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	get := func(t *testing.T, target string) leaderboardResponse {
		t.Helper()
		rec := httptest.NewRecorder()
		handleAPILeaderboard(rec, httptest.NewRequest(http.MethodGet, target, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d: %s", target, rec.Code, rec.Body.String())
		}
		var resp leaderboardResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return resp
	}

	byFiles := get(t, "/api/leaderboard?sort=files")
	if byFiles.Users[0].UserID != 1 || byFiles.Sort != "files" {
		t.Fatalf("sort=files → %+v", byFiles)
	}
	bySize := get(t, "/api/leaderboard?sort=size")
	if bySize.Users[0].UserID != 2 || bySize.Sort != "size" {
		t.Fatalf("sort=size → %+v", bySize)
	}
	// An unknown sort falls back rather than erroring.
	if fallback := get(t, "/api/leaderboard?sort=bogus"); fallback.Sort != "files" {
		t.Fatalf("bogus sort → %q, want files", fallback.Sort)
	}

	search := get(t, "/api/leaderboard?q=alph")
	if len(search.Users) != 1 || search.Users[0].UserID != 1 {
		t.Fatalf("search alph → %+v", search.Users)
	}
	if search.Query != "alph" || search.Matched != 1 || search.Total != 3 {
		t.Fatalf("search metadata wrong: %+v", search)
	}

	if cjk := get(t, "/api/leaderboard?q=%E5%B0%8F"); len(cjk.Users) != 1 || cjk.Users[0].UserID != 3 {
		t.Fatalf("cjk search → %+v", cjk.Users)
	}
	if none := get(t, "/api/leaderboard?q=zzzz"); len(none.Users) != 0 || none.Matched != 0 {
		t.Fatalf("empty search → %+v", none)
	}

	// Search and sort compose.
	both := get(t, "/api/leaderboard?q=a&sort=size")
	if len(both.Users) < 2 || both.Users[0].UserID != 2 {
		t.Fatalf("search+sort → %+v", both.Users)
	}
}

func TestHandleAPILeaderboardRejectsLongQuery(t *testing.T) {
	setupUserTestDB(t)
	long := strings.Repeat("x", maxSearchQueryLen+1)
	rec := httptest.NewRecorder()
	handleAPILeaderboard(rec, httptest.NewRequest(http.MethodGet, "/api/leaderboard?q="+long, nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
