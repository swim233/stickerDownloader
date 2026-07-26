package lib

import (
	"sync/atomic"
	"time"
)

type RuntimeErrorCategory int

const (
	RuntimeErrorPanic RuntimeErrorCategory = iota
	RuntimeErrorRequest
	RuntimeErrorDownload
	RuntimeErrorNotification
)

type RuntimeStatus struct {
	StartTime          time.Time
	SingleDownload     atomic.Int64
	PackDownload       atomic.Int64
	HTTPSingleDownload atomic.Int64
	HTTPPackDownload   atomic.Int64
	Errors             atomic.Int64
	PanicErrors        atomic.Int64
	RequestErrors      atomic.Int64
	DownloadErrors     atomic.Int64
	NotificationErrors atomic.Int64
	CacheHits          atomic.Int64
}

func (s *RuntimeStatus) RecordError(category RuntimeErrorCategory) {
	switch category {
	case RuntimeErrorPanic:
		s.PanicErrors.Add(1)
	case RuntimeErrorRequest:
		s.RequestErrors.Add(1)
	case RuntimeErrorDownload:
		s.DownloadErrors.Add(1)
	case RuntimeErrorNotification:
		s.NotificationErrors.Add(1)
	default:
		return
	}
	s.Errors.Add(1)
}
