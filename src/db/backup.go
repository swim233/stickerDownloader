package db

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// DatabaseDir and databaseFile mirror the layout InitDB creates.
const (
	DatabaseDir  = "db"
	databaseFile = "data.db"
)

// requiredTables are the tables a snapshot must contain to be a plausible
// StickerDownloader database rather than some unrelated SQLite file.
var requiredTables = []string{"user_data", "sticker_data"}

// DatabasePath returns the location of the live database file.
func DatabasePath() string {
	return filepath.Join(DatabaseDir, databaseFile)
}

// ExportSnapshot writes a consistent copy of the database to dest.
//
// VACUUM INTO rather than a file copy: the live file may have writes in
// flight, and copying it byte-for-byte can capture a torn page.
func ExportSnapshot(dest string) error {
	if DB == nil {
		return errors.New("数据库未初始化")
	}
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清理导出目标: %w", err)
	}
	if err := DB.Exec("VACUUM INTO ?", dest).Error; err != nil {
		return fmt.Errorf("导出数据库快照: %w", err)
	}
	return nil
}

// ValidateSnapshot reports whether path is a SQLite database holding this
// application's tables. It opens a separate connection so the live database
// is never touched.
func ValidateSnapshot(path string) error {
	candidate, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("无法打开上传的文件，可能不是 SQLite 数据库: %w", err)
	}
	defer closeGorm(candidate)

	// Opening is lazy; this is the first statement that really reads the file.
	var count int64
	if err := candidate.Raw("SELECT count(*) FROM sqlite_master").Scan(&count).Error; err != nil {
		return fmt.Errorf("无法读取上传的文件，可能不是 SQLite 数据库: %w", err)
	}
	migrator := candidate.Migrator()
	for _, table := range requiredTables {
		if !migrator.HasTable(table) {
			return fmt.Errorf("上传的数据库缺少 %s 表，不是 StickerDownloader 的数据库", table)
		}
	}
	return nil
}

func closeGorm(handle *gorm.DB) {
	if handle == nil {
		return
	}
	if sqlDB, err := handle.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

// Close releases the live database handle. Used before the file underneath
// it is replaced.
func Close() {
	closeGorm(DB)
}

// ReplaceDatabase swaps the live database file for the one at src, after
// backing up the current file. The caller must restart the worker
// afterwards: every open handle still points at the replaced file.
//
// Returns the path of the backup that was taken.
func ReplaceDatabase(src string, now time.Time) (string, error) {
	if err := ValidateSnapshot(src); err != nil {
		return "", err
	}

	target := DatabasePath()
	backup := target + ".bak-" + now.Format("20060102-150405")
	if _, err := os.Stat(target); err == nil {
		if err := copyFile(target, backup); err != nil {
			return "", fmt.Errorf("备份现有数据库: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("检查现有数据库: %w", err)
	}

	// Drop the handle before touching the file, so no writes land in the
	// journal of a database that is about to disappear.
	Close()

	// The write-ahead log and shared-memory files belong to the old database;
	// leaving them next to a new one risks corruption.
	for _, sidecar := range []string{target + "-wal", target + "-shm", target + "-journal"} {
		if err := os.Remove(sidecar); err != nil && !os.IsNotExist(err) {
			return backup, fmt.Errorf("清理数据库附属文件: %w", err)
		}
	}
	if err := os.Rename(src, target); err != nil {
		// Rename fails across filesystems; fall back to a copy.
		if err := copyFile(src, target); err != nil {
			return backup, fmt.Errorf("写入新数据库: %w", err)
		}
		_ = os.Remove(src)
	}
	return backup, nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
