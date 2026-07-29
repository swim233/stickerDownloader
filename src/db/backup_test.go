package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// useTempWorkingDir runs the test inside a scratch directory so the relative
// "db/data.db" layout can be exercised for real.
func useTempWorkingDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	return dir
}

// openLiveDB creates a real database at db/data.db and points DB at it.
func openLiveDB(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll(DatabaseDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	oldDB := DB
	database, err := gorm.Open(sqlite.Open(DatabasePath()), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := database.AutoMigrate(&UserData{}, &StickerData{}, &DownloadRecordData{}, &BannedUserData{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	DB = database
	t.Cleanup(func() {
		closeGorm(DB)
		DB = oldDB
	})
}

func countUsersIn(t *testing.T, path string) int64 {
	t.Helper()
	handle, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer closeGorm(handle)
	var total int64
	if err := handle.Model(&UserData{}).Count(&total).Error; err != nil {
		t.Fatalf("count in %s: %v", path, err)
	}
	return total
}

func TestExportSnapshotCapturesData(t *testing.T) {
	useTempWorkingDir(t)
	openLiveDB(t)
	for i := range 3 {
		if err := DB.Create(&UserData{UserID: int64(i + 1), FirstName: "U"}).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	dest := filepath.Join(DatabaseDir, "snapshot.db")
	if err := ExportSnapshot(dest); err != nil {
		t.Fatalf("export: %v", err)
	}
	info, err := os.Stat(dest)
	if err != nil || info.Size() == 0 {
		t.Fatalf("snapshot missing or empty: %v", err)
	}
	if got := countUsersIn(t, dest); got != 3 {
		t.Fatalf("snapshot holds %d users, want 3", got)
	}

	// Exporting again over an existing file must succeed.
	if err := ExportSnapshot(dest); err != nil {
		t.Fatalf("second export: %v", err)
	}
}

func TestValidateSnapshot(t *testing.T) {
	dir := t.TempDir()

	notSQLite := filepath.Join(dir, "junk.db")
	if err := os.WriteFile(notSQLite, []byte("this is definitely not a database"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := ValidateSnapshot(notSQLite); err == nil {
		t.Fatal("a text file must not validate")
	}

	// A valid SQLite file that isn't ours.
	wrongSchema := filepath.Join(dir, "other.db")
	other, err := gorm.Open(sqlite.Open(wrongSchema), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := other.Exec("CREATE TABLE unrelated (id integer)").Error; err != nil {
		t.Fatalf("create: %v", err)
	}
	closeGorm(other)
	if err := ValidateSnapshot(wrongSchema); err == nil {
		t.Fatal("an unrelated SQLite database must not validate")
	}

	// A real one.
	good := filepath.Join(dir, "good.db")
	ours, err := gorm.Open(sqlite.Open(good), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := ours.AutoMigrate(&UserData{}, &StickerData{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	closeGorm(ours)
	if err := ValidateSnapshot(good); err != nil {
		t.Fatalf("a real database must validate: %v", err)
	}
}

func TestReplaceDatabaseBacksUpAndSwaps(t *testing.T) {
	useTempWorkingDir(t)
	openLiveDB(t)
	// Two users in the live database...
	for i := range 2 {
		if err := DB.Create(&UserData{UserID: int64(i + 1), FirstName: "live"}).Error; err != nil {
			t.Fatalf("seed live: %v", err)
		}
	}

	// ...and five in the replacement.
	incoming := filepath.Join(t.TempDir(), "incoming.db")
	replacement, err := gorm.Open(sqlite.Open(incoming), &gorm.Config{})
	if err != nil {
		t.Fatalf("open incoming: %v", err)
	}
	if err := replacement.AutoMigrate(&UserData{}, &StickerData{}, &DownloadRecordData{}, &BannedUserData{}); err != nil {
		t.Fatalf("migrate incoming: %v", err)
	}
	for i := range 5 {
		if err := replacement.Create(&UserData{UserID: int64(100 + i), FirstName: "imported"}).Error; err != nil {
			t.Fatalf("seed incoming: %v", err)
		}
	}
	closeGorm(replacement)

	backup, err := ReplaceDatabase(incoming, time.Date(2026, 7, 28, 13, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("replace: %v", err)
	}

	if backup == "" {
		t.Fatal("no backup path returned")
	}
	if got := countUsersIn(t, backup); got != 2 {
		t.Fatalf("backup holds %d users, want the original 2", got)
	}
	if got := countUsersIn(t, DatabasePath()); got != 5 {
		t.Fatalf("live database holds %d users, want the imported 5", got)
	}
	if _, err := os.Stat(incoming); !os.IsNotExist(err) {
		t.Fatal("the staged upload should have been consumed")
	}
}

func TestReplaceDatabaseRejectsInvalidFile(t *testing.T) {
	useTempWorkingDir(t)
	openLiveDB(t)
	if err := DB.Create(&UserData{UserID: 1, FirstName: "live"}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	junk := filepath.Join(t.TempDir(), "junk.db")
	if err := os.WriteFile(junk, []byte("nope"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := ReplaceDatabase(junk, time.Now()); err == nil {
		t.Fatal("expected rejection of an invalid file")
	}
	// The live database must be untouched, and still usable.
	if got := countUsersIn(t, DatabasePath()); got != 1 {
		t.Fatalf("live database was disturbed: %d users", got)
	}
	var total int64
	if err := DB.Model(&UserData{}).Count(&total).Error; err != nil {
		t.Fatalf("live handle broken after rejected import: %v", err)
	}
}

func TestReplaceDatabaseClearsWALSidecars(t *testing.T) {
	useTempWorkingDir(t)
	openLiveDB(t)

	// Simulate leftovers from the outgoing database.
	for _, sidecar := range []string{DatabasePath() + "-wal", DatabasePath() + "-shm"} {
		if err := os.WriteFile(sidecar, []byte("stale"), 0600); err != nil {
			t.Fatalf("write sidecar: %v", err)
		}
	}

	incoming := filepath.Join(t.TempDir(), "incoming.db")
	replacement, err := gorm.Open(sqlite.Open(incoming), &gorm.Config{})
	if err != nil {
		t.Fatalf("open incoming: %v", err)
	}
	if err := replacement.AutoMigrate(&UserData{}, &StickerData{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	closeGorm(replacement)

	if _, err := ReplaceDatabase(incoming, time.Now()); err != nil {
		t.Fatalf("replace: %v", err)
	}
	for _, sidecar := range []string{DatabasePath() + "-wal", DatabasePath() + "-shm"} {
		if _, err := os.Stat(sidecar); !os.IsNotExist(err) {
			t.Fatalf("stale sidecar %s survived the swap", sidecar)
		}
	}
}
