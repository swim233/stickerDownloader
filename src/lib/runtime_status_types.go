package lib

import "time"

type RuntimeStatus struct {
	StartTime          time.Time
	SingleDownload     int
	PackDownload       int
	HTTPSingleDownload int
	HTTPPackDownload   int
	Errors             int
	CacheHits          int
	HitPercentage      float64
}
