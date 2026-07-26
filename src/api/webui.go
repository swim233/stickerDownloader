package api

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"time"

	"github.com/swim233/StickerDownloader/lib"
	"github.com/swim233/StickerDownloader/utils"
)

//go:embed webui/index.html
var webuiIndexHTML []byte

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(webuiIndexHTML)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, utils.CurrentRuntimeStatusReport(time.Now()))
}

type historyResponse struct {
	Capacity int                  `json:"capacity"`
	Records  []lib.DownloadRecord `json:"records"`
}

func handleAPIHistory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, historyResponse{
		Capacity: utils.DownloadHistory.Capacity(),
		Records:  utils.DownloadHistory.Recent(),
	})
}
