package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	db "github.com/swim233/StickerDownloader/db"
	"github.com/swim233/StickerDownloader/lib"
	"gorm.io/gorm"
)

// useLiveDatabase sets up a real db/data.db under a scratch working
// directory, matching how the worker runs.
func useLiveDatabase(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := os.MkdirAll(db.DatabaseDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	oldDB := db.DB
	database, err := gorm.Open(sqlite.Open(db.DatabasePath()), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := database.AutoMigrate(&db.UserData{}, &db.StickerData{}, &db.DownloadRecordData{}, &db.BannedUserData{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.DB = database
	t.Cleanup(func() {
		db.Close()
		db.DB = oldDB
		_ = os.Chdir(old)
	})
}

// buildDatabaseFile writes a valid application database with n users.
func buildDatabaseFile(t *testing.T, path string, users int) {
	t.Helper()
	handle, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := handle.AutoMigrate(&db.UserData{}, &db.StickerData{}, &db.DownloadRecordData{}, &db.BannedUserData{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for i := range users {
		if err := handle.Create(&db.UserData{UserID: int64(500 + i), FirstName: "imported"}).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	if sqlDB, err := handle.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

func uploadRequest(t *testing.T, field, filename string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/db/import", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func drainRestartRequests() {
	for {
		select {
		case <-lib.RestartRequests():
		default:
			return
		}
	}
}

func TestHandleAPIDatabaseInfo(t *testing.T) {
	useLiveDatabase(t)
	if err := db.DB.Create(&db.UserData{UserID: 7, FirstName: "A"}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	rec := httptest.NewRecorder()
	handleAPIDatabaseInfo(rec, httptest.NewRequest(http.MethodGet, "/api/db/info", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp databaseInfoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Users != 1 || resp.SizeBytes == 0 || resp.MaxUpload != maxImportBytes {
		t.Fatalf("info = %+v", resp)
	}
	if !filepath.IsAbs(resp.Path) {
		t.Fatalf("path %q should be absolute", resp.Path)
	}
}

func TestHandleAPIDatabaseExport(t *testing.T) {
	useLiveDatabase(t)
	if err := db.DB.Create(&db.UserData{UserID: 7, FirstName: "A"}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	rec := httptest.NewRecorder()
	handleAPIDatabaseExport(rec, httptest.NewRequest(http.MethodGet, "/api/db/export", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Fatalf("Content-Type = %s", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); !bytes.Contains([]byte(cd), []byte("attachment")) {
		t.Fatalf("Content-Disposition = %s", cd)
	}
	body := rec.Body.Bytes()
	if !bytes.HasPrefix(body, []byte("SQLite format 3")) {
		t.Fatalf("export is not a SQLite file, starts with %q", body[:min(16, len(body))])
	}

	// The snapshot must be restorable, and the temp file must not linger.
	restored := filepath.Join(t.TempDir(), "restored.db")
	if err := os.WriteFile(restored, body, 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := db.ValidateSnapshot(restored); err != nil {
		t.Fatalf("exported snapshot does not validate: %v", err)
	}
	entries, err := os.ReadDir(db.DatabaseDir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, entry := range entries {
		if len(entry.Name()) > 7 && entry.Name()[:7] == "export-" {
			t.Fatalf("temporary export file %s was left behind", entry.Name())
		}
	}
}

func TestHandleAPIDatabaseImportReplacesAndRestarts(t *testing.T) {
	useLiveDatabase(t)
	drainRestartRequests()
	if err := db.DB.Create(&db.UserData{UserID: 1, FirstName: "original"}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	incoming := filepath.Join(t.TempDir(), "incoming.db")
	buildDatabaseFile(t, incoming, 4)
	content, err := os.ReadFile(incoming)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	rec := httptest.NewRecorder()
	handleAPIDatabaseImport(rec, uploadRequest(t, "database", "backup.db", content))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	var resp databaseImportResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.OK || resp.Backup == "" || !resp.Restart {
		t.Fatalf("response = %+v", resp)
	}
	if _, err := os.Stat(resp.Backup); err != nil {
		t.Fatalf("backup %s missing: %v", resp.Backup, err)
	}

	// The live file is now the uploaded one.
	if err := db.ValidateSnapshot(db.DatabasePath()); err != nil {
		t.Fatalf("replaced database invalid: %v", err)
	}
	handle, err := gorm.Open(sqlite.Open(db.DatabasePath()), &gorm.Config{})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	var total int64
	_ = handle.Model(&db.UserData{}).Count(&total).Error
	if sqlDB, err := handle.DB(); err == nil {
		_ = sqlDB.Close()
	}
	if total != 4 {
		t.Fatalf("live database has %d users, want the imported 4", total)
	}

	select {
	case <-lib.RestartRequests():
	default:
		t.Fatal("import did not request a restart")
	}
}

func TestHandleAPIDatabaseImportRejectsBadUploads(t *testing.T) {
	cases := []struct {
		name    string
		field   string
		content []byte
		status  int
	}{
		{"not a database", "database", []byte("just some text"), http.StatusBadRequest},
		{"empty file", "database", []byte{}, http.StatusBadRequest},
		{"wrong field name", "file", []byte("x"), http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useLiveDatabase(t)
			drainRestartRequests()
			if err := db.DB.Create(&db.UserData{UserID: 1, FirstName: "original"}).Error; err != nil {
				t.Fatalf("seed: %v", err)
			}

			rec := httptest.NewRecorder()
			handleAPIDatabaseImport(rec, uploadRequest(t, tc.field, "x.db", tc.content))
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tc.status, rec.Body.String())
			}

			// A rejected import must leave the running database usable.
			var total int64
			if err := db.DB.Model(&db.UserData{}).Count(&total).Error; err != nil {
				t.Fatalf("live database broken after rejected import: %v", err)
			}
			if total != 1 {
				t.Fatalf("live database has %d users, want the original 1", total)
			}
			select {
			case <-lib.RestartRequests():
				t.Fatal("a rejected import must not request a restart")
			default:
			}
		})
	}
}

func TestHandleAPIDatabaseImportRejectsGet(t *testing.T) {
	rec := httptest.NewRecorder()
	handleAPIDatabaseImport(rec, httptest.NewRequest(http.MethodGet, "/api/db/import", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestHandleAPIDatabaseImportRejectsOversizedUpload(t *testing.T) {
	useLiveDatabase(t)
	drainRestartRequests()

	// Stream past the limit without allocating it: the reader is capped.
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("database", "big.db")
	_, _ = io.CopyN(part, zeroReader{}, maxImportBytes+1024)
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/db/import", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	handleAPIDatabaseImport(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatal("an oversized upload must be rejected")
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}
