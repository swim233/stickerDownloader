package lib

import (
	"sync"
	"time"
)

// DefaultDownloadHistorySize is used when no valid capacity is configured.
const DefaultDownloadHistorySize = 10

const (
	DownloadSourceBot  = "bot"
	DownloadSourceHTTP = "http"

	DownloadKindSingle = "single"
	DownloadKindPack   = "pack"
)

// DownloadRecord describes one completed download for the WebUI history view.
type DownloadRecord struct {
	Time          time.Time `json:"time"`
	Source        string    `json:"source"`
	Kind          string    `json:"kind"`
	UserID        int64     `json:"user_id,omitempty"`
	UserName      string    `json:"user_name,omitempty"`
	DisplayName   string    `json:"display_name,omitempty"`
	SetName       string    `json:"set_name,omitempty"`
	SetTitle      string    `json:"set_title,omitempty"`
	StickerFileID string    `json:"sticker_file_id,omitempty"`
	StickerEmoji  string    `json:"sticker_emoji,omitempty"`
	Format        string    `json:"format"`
	FileCount     int       `json:"file_count"`
	FileSize      int64     `json:"file_size"`
	CacheHit      bool      `json:"cache_hit,omitempty"`
}

// DownloadHistory keeps the most recent download records in memory.
type DownloadHistory struct {
	mu       sync.Mutex
	capacity int
	records  []DownloadRecord
	persist  func(DownloadRecord)
}

func NewDownloadHistory(capacity int) *DownloadHistory {
	if capacity <= 0 {
		capacity = DefaultDownloadHistorySize
	}
	return &DownloadHistory{capacity: capacity}
}

// SetCapacity resizes the history, dropping the oldest records if needed.
func (h *DownloadHistory) SetCapacity(capacity int) {
	if capacity <= 0 {
		capacity = DefaultDownloadHistorySize
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.capacity = capacity
	if len(h.records) > capacity {
		h.records = append([]DownloadRecord(nil), h.records[len(h.records)-capacity:]...)
	}
}

func (h *DownloadHistory) Capacity() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.capacity
}

// SetPersist installs a hook called with every record passed to Add.
func (h *DownloadHistory) SetPersist(persist func(DownloadRecord)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.persist = persist
}

// Seed replaces the in-memory records with previously persisted ones
// (oldest first) without invoking the persist hook.
func (h *DownloadHistory) Seed(records []DownloadRecord) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(records) > h.capacity {
		records = records[len(records)-h.capacity:]
	}
	h.records = append([]DownloadRecord(nil), records...)
}

// Add appends a record, evicting the oldest once the capacity is reached.
func (h *DownloadHistory) Add(record DownloadRecord) {
	if record.Time.IsZero() {
		record.Time = time.Now()
	}
	h.mu.Lock()
	h.records = append(h.records, record)
	if len(h.records) > h.capacity {
		h.records = h.records[len(h.records)-h.capacity:]
	}
	persist := h.persist
	h.mu.Unlock()
	if persist != nil {
		persist(record)
	}
}

// Recent returns a copy of the stored records, newest first.
func (h *DownloadHistory) Recent() []DownloadRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	recent := make([]DownloadRecord, len(h.records))
	for i, record := range h.records {
		recent[len(h.records)-1-i] = record
	}
	return recent
}
