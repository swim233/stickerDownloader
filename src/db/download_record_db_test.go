package db

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/swim233/StickerDownloader/lib"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	old := DB
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := database.AutoMigrate(&DownloadRecordData{}, &BannedUserData{}, &UserData{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	DB = database
	t.Cleanup(func() { DB = old })
}

func TestSaveDownloadRecordPrunesToKeep(t *testing.T) {
	setupTestDB(t)
	for i := range 12 {
		record := lib.DownloadRecord{
			Time:    time.Unix(int64(i), 0),
			Source:  lib.DownloadSourceBot,
			Kind:    lib.DownloadKindPack,
			SetName: fmt.Sprintf("set%d", i),
			Format:  "webp",
		}
		if err := SaveDownloadRecord(record, 10); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}

	var count int64
	if err := DB.Model(&DownloadRecordData{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 10 {
		t.Fatalf("count = %d, want 10", count)
	}

	records, err := LoadRecentDownloadRecords(10)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(records) != 10 {
		t.Fatalf("len = %d, want 10", len(records))
	}
	if records[0].SetName != "set2" || records[9].SetName != "set11" {
		t.Fatalf("records span %s..%s, want set2..set11 oldest first", records[0].SetName, records[9].SetName)
	}
}

func TestDownloadRecordRoundTrip(t *testing.T) {
	setupTestDB(t)
	want := lib.DownloadRecord{
		Time:          time.Unix(1700000000, 0).UTC(),
		Source:        lib.DownloadSourceBot,
		Kind:          lib.DownloadKindSingle,
		UserID:        42,
		UserName:      "tester",
		DisplayName:   "Test User",
		SetName:       "cats",
		SetTitle:      "Cats!",
		StickerFileID: "file-id-1",
		StickerEmoji:  "😺",
		Format:        "png",
		FileCount:     1,
		FileSize:      2048,
		CacheHit:      true,
	}
	if err := SaveDownloadRecord(want, 10); err != nil {
		t.Fatalf("save: %v", err)
	}
	records, err := LoadRecentDownloadRecords(10)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len = %d, want 1", len(records))
	}
	got := records[0]
	got.Time = got.Time.UTC()
	if got != want {
		t.Fatalf("round trip mismatch:\ngot  %+v\nwant %+v", got, want)
	}
}
