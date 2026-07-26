package lib

import (
	"testing"
	"time"
)

func TestDownloadHistoryEvictsOldest(t *testing.T) {
	h := NewDownloadHistory(3)
	for i := range 5 {
		h.Add(DownloadRecord{SetName: string(rune('a' + i)), Time: time.Unix(int64(i), 0)})
	}
	recent := h.Recent()
	if len(recent) != 3 {
		t.Fatalf("len = %d, want 3", len(recent))
	}
	if recent[0].SetName != "e" || recent[2].SetName != "c" {
		t.Fatalf("recent = %v, want newest first e..c", recent)
	}
}

func TestDownloadHistoryDefaultCapacity(t *testing.T) {
	h := NewDownloadHistory(0)
	if got := h.Capacity(); got != DefaultDownloadHistorySize {
		t.Fatalf("Capacity = %d, want %d", got, DefaultDownloadHistorySize)
	}
}

func TestDownloadHistorySetCapacityDropsOldest(t *testing.T) {
	h := NewDownloadHistory(5)
	for i := range 5 {
		h.Add(DownloadRecord{SetName: string(rune('a' + i)), Time: time.Unix(int64(i), 0)})
	}
	h.SetCapacity(2)
	recent := h.Recent()
	if len(recent) != 2 {
		t.Fatalf("len = %d, want 2", len(recent))
	}
	if recent[0].SetName != "e" || recent[1].SetName != "d" {
		t.Fatalf("recent = %v, want e,d", recent)
	}
}

func TestDownloadHistorySeedSkipsPersistAndTrims(t *testing.T) {
	h := NewDownloadHistory(2)
	persisted := 0
	h.SetPersist(func(DownloadRecord) { persisted++ })

	h.Seed([]DownloadRecord{
		{SetName: "a", Time: time.Unix(0, 0)},
		{SetName: "b", Time: time.Unix(1, 0)},
		{SetName: "c", Time: time.Unix(2, 0)},
	})
	if persisted != 0 {
		t.Fatalf("persisted = %d, want 0 after Seed", persisted)
	}
	recent := h.Recent()
	if len(recent) != 2 || recent[0].SetName != "c" || recent[1].SetName != "b" {
		t.Fatalf("recent = %v, want c,b", recent)
	}

	h.Add(DownloadRecord{SetName: "d"})
	if persisted != 1 {
		t.Fatalf("persisted = %d, want 1 after Add", persisted)
	}
	if got := h.Recent()[0].SetName; got != "d" {
		t.Fatalf("newest = %s, want d", got)
	}
}

func TestDownloadHistoryAddStampsTime(t *testing.T) {
	h := NewDownloadHistory(1)
	h.Add(DownloadRecord{SetName: "a"})
	if h.Recent()[0].Time.IsZero() {
		t.Fatal("expected Add to stamp a zero Time")
	}
}
