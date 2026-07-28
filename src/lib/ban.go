package lib

import "time"

// BanRecord describes a banned user for enforcement and the WebUI list.
type BanRecord struct {
	UserID   int64     `json:"user_id"`
	Reason   string    `json:"reason,omitempty"`
	Silent   bool      `json:"silent,omitempty"`
	BannedAt time.Time `json:"banned_at"`
}
