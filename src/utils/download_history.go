package utils

import "github.com/swim233/StickerDownloader/lib"

// DownloadHistory keeps the most recent downloads for the WebUI history view.
// Capacity is applied from config at worker startup via SetCapacity.
var DownloadHistory = lib.NewDownloadHistory(lib.DefaultDownloadHistorySize)
