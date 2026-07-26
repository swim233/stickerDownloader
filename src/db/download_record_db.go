package db

import (
	"time"

	"github.com/swim233/StickerDownloader/lib"
)

// DownloadRecordData persists WebUI download-history rows across restarts.
type DownloadRecordData struct {
	ID            uint `gorm:"primaryKey"`
	Time          time.Time
	Source        string
	Kind          string
	UserID        int64
	UserName      string
	DisplayName   string
	SetName       string
	SetTitle      string
	StickerFileID string
	StickerEmoji  string
	Format        string
	FileCount     int
	FileSize      int64
	CacheHit      bool
}

func (d DownloadRecordData) toLib() lib.DownloadRecord {
	return lib.DownloadRecord{
		Time:          d.Time,
		Source:        d.Source,
		Kind:          d.Kind,
		UserID:        d.UserID,
		UserName:      d.UserName,
		DisplayName:   d.DisplayName,
		SetName:       d.SetName,
		SetTitle:      d.SetTitle,
		StickerFileID: d.StickerFileID,
		StickerEmoji:  d.StickerEmoji,
		Format:        d.Format,
		FileCount:     d.FileCount,
		FileSize:      d.FileSize,
		CacheHit:      d.CacheHit,
	}
}

// SaveDownloadRecord appends a record and prunes rows beyond the newest keep.
func SaveDownloadRecord(record lib.DownloadRecord, keep int) error {
	row := DownloadRecordData{
		Time:          record.Time,
		Source:        record.Source,
		Kind:          record.Kind,
		UserID:        record.UserID,
		UserName:      record.UserName,
		DisplayName:   record.DisplayName,
		SetName:       record.SetName,
		SetTitle:      record.SetTitle,
		StickerFileID: record.StickerFileID,
		StickerEmoji:  record.StickerEmoji,
		Format:        record.Format,
		FileCount:     record.FileCount,
		FileSize:      record.FileSize,
		CacheHit:      record.CacheHit,
	}
	if err := DB.Create(&row).Error; err != nil {
		return err
	}
	if keep <= 0 {
		keep = lib.DefaultDownloadHistorySize
	}
	newest := DB.Model(&DownloadRecordData{}).Select("id").Order("id DESC").Limit(keep)
	return DB.Where("id NOT IN (?)", newest).Delete(&DownloadRecordData{}).Error
}

// LoadRecentDownloadRecords returns up to n persisted records, oldest first.
func LoadRecentDownloadRecords(n int) ([]lib.DownloadRecord, error) {
	if n <= 0 {
		n = lib.DefaultDownloadHistorySize
	}
	var rows []DownloadRecordData
	if err := DB.Order("id DESC").Limit(n).Find(&rows).Error; err != nil {
		return nil, err
	}
	records := make([]lib.DownloadRecord, len(rows))
	for i, row := range rows {
		records[len(rows)-1-i] = row.toLib()
	}
	return records, nil
}
