package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	db "github.com/swim233/StickerDownloader/db"
	"github.com/swim233/StickerDownloader/lib"
	logger "github.com/swim233/StickerDownloader/logger"
)

// maxImportBytes bounds an uploaded database. The real one is a couple of
// megabytes; this leaves generous headroom without letting an upload fill
// the disk.
const maxImportBytes = 128 << 20

type databaseInfoResponse struct {
	Path       string `json:"path"`
	SizeBytes  int64  `json:"size_bytes"`
	ModifiedAt string `json:"modified_at,omitempty"`
	Users      int64  `json:"users"`
	Bans       int    `json:"bans"`
	Records    int64  `json:"records"`
	MaxUpload  int64  `json:"max_upload_bytes"`
}

func handleAPIDatabaseInfo(w http.ResponseWriter, r *http.Request) {
	resp := databaseInfoResponse{Path: db.DatabasePath(), MaxUpload: maxImportBytes}
	if absolute, err := filepath.Abs(resp.Path); err == nil {
		resp.Path = absolute
	}
	if info, err := os.Stat(db.DatabasePath()); err == nil {
		resp.SizeBytes = info.Size()
		resp.ModifiedAt = info.ModTime().Format(time.RFC3339)
	}
	if users, err := db.CountUsers(); err == nil {
		resp.Users = users
	}
	resp.Bans = len(db.ListBans())
	if records, err := db.CountDownloadRecords(); err == nil {
		resp.Records = records
	}
	writeJSON(w, resp)
}

// handleAPIDatabaseExport streams a consistent snapshot of the database.
func handleAPIDatabaseExport(w http.ResponseWriter, r *http.Request) {
	temp, err := os.CreateTemp(db.DatabaseDir, "export-*.db")
	if err != nil {
		logger.Error("创建导出临时文件失败: %s", err)
		writeJSONError(w, http.StatusInternalServerError, "导出失败")
		return
	}
	snapshot := temp.Name()
	// VACUUM INTO insists on creating the file itself.
	_ = temp.Close()
	defer os.Remove(snapshot)

	if err := db.ExportSnapshot(snapshot); err != nil {
		logger.Error("导出数据库失败: %s", err)
		writeJSONError(w, http.StatusInternalServerError, "导出失败: "+err.Error())
		return
	}

	file, err := os.Open(snapshot)
	if err != nil {
		logger.Error("读取数据库快照失败: %s", err)
		writeJSONError(w, http.StatusInternalServerError, "导出失败")
		return
	}
	defer file.Close()

	name := fmt.Sprintf("stickerdownloader-%s.db", time.Now().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	if info, err := file.Stat(); err == nil {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	}
	if _, err := io.Copy(w, file); err != nil {
		logger.Warn("发送数据库快照失败: %s", err)
	}
	logger.Info("WebUI 导出了数据库快照")
}

type databaseImportResponse struct {
	OK       bool   `json:"ok"`
	Backup   string `json:"backup"`
	Restart  bool   `json:"restart"`
	SizeRead int64  `json:"size_read"`
}

// handleAPIDatabaseImport replaces the live database with an uploaded one.
// The current database is backed up first and the worker restarts, because
// every open handle still refers to the file that was swapped out.
func handleAPIDatabaseImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "仅支持 POST")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImportBytes)
	upload, _, err := r.FormFile("database")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "请选择要导入的数据库文件")
		return
	}
	defer upload.Close()

	staged, err := os.CreateTemp(db.DatabaseDir, "import-*.db")
	if err != nil {
		logger.Error("创建导入临时文件失败: %s", err)
		writeJSONError(w, http.StatusInternalServerError, "导入失败")
		return
	}
	stagedPath := staged.Name()
	// Removed on every failure path; on success the rename consumes it.
	defer os.Remove(stagedPath)

	written, err := io.Copy(staged, io.LimitReader(upload, maxImportBytes+1))
	if closeErr := staged.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		logger.Warn("接收上传的数据库失败: %s", err)
		writeJSONError(w, http.StatusBadRequest, "接收上传文件失败")
		return
	}
	if written > maxImportBytes {
		writeJSONError(w, http.StatusRequestEntityTooLarge, "数据库文件过大")
		return
	}
	if written == 0 {
		writeJSONError(w, http.StatusBadRequest, "上传的文件为空")
		return
	}

	// Validate before anything destructive happens.
	if err := db.ValidateSnapshot(stagedPath); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	backup, err := db.ReplaceDatabase(stagedPath, time.Now())
	if err != nil {
		logger.Error("导入数据库失败: %s", err)
		message := "导入失败: " + err.Error()
		if backup != "" {
			message += "（原数据库已备份至 " + backup + "）"
		}
		writeJSONError(w, http.StatusInternalServerError, message)
		return
	}

	logger.Info("WebUI 导入了数据库，原库备份至 %s", backup)
	restarted := lib.RequestRestart("WebUI 导入数据库")
	writeJSON(w, databaseImportResponse{OK: true, Backup: backup, Restart: restarted, SizeRead: written})
}
