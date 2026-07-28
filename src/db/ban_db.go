package db

import (
	"sort"
	"sync"
	"time"

	"github.com/swim233/StickerDownloader/lib"
)

// BannedUserData persists bans; an in-memory cache backs per-update checks.
//
// The uniqueIndex matters as much as the primary key: SQLite cannot add a
// primary key to a table that already exists, but AutoMigrate does create the
// index, so deployments upgraded in place still get the uniqueness guarantee.
type BannedUserData struct {
	UserID   int64 `gorm:"primaryKey;autoIncrement:false;uniqueIndex:idx_banned_user_id"`
	Reason   string
	Silent   bool
	BannedAt time.Time
}

func (b BannedUserData) toLib() lib.BanRecord {
	return lib.BanRecord{UserID: b.UserID, Reason: b.Reason, Silent: b.Silent, BannedAt: b.BannedAt}
}

var banCache = struct {
	sync.RWMutex
	users map[int64]lib.BanRecord
}{users: map[int64]lib.BanRecord{}}

// DedupeBans collapses duplicate ban rows left behind by older versions,
// keeping the most recent entry per user. Called from InitDB before
// AutoMigrate adds the unique index, which would otherwise fail on an
// upgraded database that still holds duplicates.
func DedupeBans() (int64, error) {
	if !DB.Migrator().HasTable(&BannedUserData{}) {
		return 0, nil
	}
	result := DB.Exec(`DELETE FROM banned_user_data WHERE rowid NOT IN (
		SELECT MAX(rowid) FROM banned_user_data GROUP BY user_id
	)`)
	return result.RowsAffected, result.Error
}

// LoadBans fills the ban cache from the database. Call once after InitDB.
func LoadBans() error {
	var rows []BannedUserData
	// Oldest first, so a duplicate that slipped through leaves the newest
	// record in the cache.
	if err := DB.Order("banned_at ASC").Find(&rows).Error; err != nil {
		return err
	}
	banCache.Lock()
	defer banCache.Unlock()
	banCache.users = make(map[int64]lib.BanRecord, len(rows))
	for _, row := range rows {
		banCache.users[row.UserID] = row.toLib()
	}
	return nil
}

// BanUser bans (or re-bans with new details) a user.
//
// The write is an explicit update-then-insert rather than an upsert: tables
// created before this column gained its unique index have no conflict target
// for ON CONFLICT to use, and a silently duplicated row would resurrect a
// stale ban after the next restart.
func BanUser(userID int64, reason string, silent bool) error {
	row := BannedUserData{UserID: userID, Reason: reason, Silent: silent, BannedAt: time.Now()}

	result := DB.Model(&BannedUserData{}).Where("user_id = ?", userID).
		Updates(map[string]any{"reason": row.Reason, "silent": row.Silent, "banned_at": row.BannedAt})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		if err := DB.Create(&row).Error; err != nil {
			return err
		}
	}

	banCache.Lock()
	banCache.users[userID] = row.toLib()
	banCache.Unlock()
	return nil
}

// UnbanUser lifts a ban and reports whether the user was banned.
func UnbanUser(userID int64) (bool, error) {
	result := DB.Delete(&BannedUserData{}, "user_id = ?", userID)
	if result.Error != nil {
		return false, result.Error
	}
	banCache.Lock()
	delete(banCache.users, userID)
	banCache.Unlock()
	return result.RowsAffected > 0, nil
}

// GetBan returns the ban for a user, if any. Reads only the cache.
func GetBan(userID int64) (lib.BanRecord, bool) {
	banCache.RLock()
	defer banCache.RUnlock()
	ban, ok := banCache.users[userID]
	return ban, ok
}

// ListBans returns all bans, newest first.
func ListBans() []lib.BanRecord {
	banCache.RLock()
	bans := make([]lib.BanRecord, 0, len(banCache.users))
	for _, ban := range banCache.users {
		bans = append(bans, ban)
	}
	banCache.RUnlock()
	sort.Slice(bans, func(i, j int) bool { return bans[i].BannedAt.After(bans[j].BannedAt) })
	return bans
}
