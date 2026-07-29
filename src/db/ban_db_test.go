package db

import (
	"testing"

	"github.com/swim233/StickerDownloader/lib"
)

// resetBanCache isolates the package-level ban cache between tests.
func resetBanCache(t *testing.T) {
	t.Helper()
	banCache.Lock()
	old := banCache.users
	banCache.users = map[int64]lib.BanRecord{}
	banCache.Unlock()
	t.Cleanup(func() {
		banCache.Lock()
		banCache.users = old
		banCache.Unlock()
	})
}

func TestBanUnbanLifecycle(t *testing.T) {
	setupTestDB(t)
	resetBanCache(t)

	if err := BanUser(100, "spam", false); err != nil {
		t.Fatalf("ban: %v", err)
	}
	if err := BanUser(200, "", true); err != nil {
		t.Fatalf("silent ban: %v", err)
	}

	ban, ok := GetBan(100)
	if !ok || ban.Reason != "spam" || ban.Silent {
		t.Fatalf("GetBan(100) = %+v, %v", ban, ok)
	}
	ban, ok = GetBan(200)
	if !ok || !ban.Silent {
		t.Fatalf("GetBan(200) = %+v, %v; want silent ban", ban, ok)
	}
	if _, ok := GetBan(300); ok {
		t.Fatal("GetBan(300) should miss")
	}

	if bans := ListBans(); len(bans) != 2 {
		t.Fatalf("ListBans len = %d, want 2", len(bans))
	}

	existed, err := UnbanUser(100)
	if err != nil || !existed {
		t.Fatalf("unban = %v, %v; want existed", existed, err)
	}
	if _, ok := GetBan(100); ok {
		t.Fatal("user 100 still banned after unban")
	}
	existed, err = UnbanUser(100)
	if err != nil || existed {
		t.Fatalf("second unban = %v, %v; want not existed", existed, err)
	}
}

func TestBanUserUpdatesExistingBan(t *testing.T) {
	setupTestDB(t)
	resetBanCache(t)
	if err := BanUser(100, "first", false); err != nil {
		t.Fatalf("ban: %v", err)
	}
	if err := BanUser(100, "second", true); err != nil {
		t.Fatalf("re-ban: %v", err)
	}
	ban, ok := GetBan(100)
	if !ok || ban.Reason != "second" || !ban.Silent {
		t.Fatalf("GetBan = %+v, %v; want updated silent ban", ban, ok)
	}
	if bans := ListBans(); len(bans) != 1 {
		t.Fatalf("ListBans len = %d, want 1", len(bans))
	}
}

func TestLoadBansRestoresCache(t *testing.T) {
	setupTestDB(t)
	resetBanCache(t)
	if err := BanUser(100, "persisted", true); err != nil {
		t.Fatalf("ban: %v", err)
	}
	// Simulate a fresh worker: wipe the cache, then reload from the database.
	banCache.Lock()
	banCache.users = map[int64]lib.BanRecord{}
	banCache.Unlock()
	if err := LoadBans(); err != nil {
		t.Fatalf("load: %v", err)
	}
	ban, ok := GetBan(100)
	if !ok || ban.Reason != "persisted" || !ban.Silent {
		t.Fatalf("GetBan after reload = %+v, %v", ban, ok)
	}
}

// TestBanUserDoesNotDuplicateRows checks the database itself, not just the
// in-memory cache: re-banning a user must update the existing row.
func TestBanUserDoesNotDuplicateRows(t *testing.T) {
	setupTestDB(t)
	resetBanCache(t)

	if err := BanUser(100, "first", false); err != nil {
		t.Fatalf("ban: %v", err)
	}
	if err := BanUser(100, "second", true); err != nil {
		t.Fatalf("re-ban: %v", err)
	}

	var rows []BannedUserData
	if err := DB.Where("user_id = ?", 100).Find(&rows).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("%d rows for user 100, want 1 (duplicates accumulate)", len(rows))
	}
	if rows[0].Reason != "second" || !rows[0].Silent {
		t.Fatalf("row = %+v, want the updated values", rows[0])
	}

	// A fresh worker must see the updated ban, not a stale duplicate.
	banCache.Lock()
	banCache.users = map[int64]lib.BanRecord{}
	banCache.Unlock()
	if err := LoadBans(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	ban, ok := GetBan(100)
	if !ok || ban.Reason != "second" || !ban.Silent {
		t.Fatalf("after reload = %+v, want the updated ban", ban)
	}
}
