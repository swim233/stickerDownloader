package lib

import (
	"sync/atomic"
	"time"
)

type RuntimeStatus struct {
	StartTime          time.Time
	SingleDownload     atomic.Int64
	PackDownload       atomic.Int64
	HTTPSingleDownload atomic.Int64
	HTTPPackDownload   atomic.Int64
	Errors             atomic.Int64
	CacheHits          atomic.Int64
}
