package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/swim233/StickerDownloader/config"
	db "github.com/swim233/StickerDownloader/db"
	"github.com/swim233/StickerDownloader/lib"
	logger "github.com/swim233/StickerDownloader/logger"
)

const leaderboardSize = 10

type userSummary struct {
	UserID           int64  `json:"user_id"`
	UserName         string `json:"user_name,omitempty"`
	DisplayName      string `json:"display_name,omitempty"`
	DownloadFiles    int    `json:"download_files"`
	DownloadFileSize int64  `json:"download_file_size"`
}

type userDetail struct {
	userSummary
	Found              bool   `json:"found"`
	CreateTime         string `json:"create_time,omitempty"`
	RecentDownloadTime string `json:"recent_download_time,omitempty"`
	Language           string `json:"language,omitempty"`
	Rank               int64  `json:"rank,omitempty"`
	IsOwner            bool   `json:"is_owner,omitempty"`
}

type userResponse struct {
	User        userDetail     `json:"user"`
	Ban         *lib.BanRecord `json:"ban,omitempty"`
	Leaderboard []userSummary  `json:"leaderboard"`
}

func summarizeUser(user db.UserData) userSummary {
	return userSummary{
		UserID:           user.UserID,
		UserName:         user.UserName,
		DisplayName:      strings.TrimSpace(user.FirstName + " " + user.LastName),
		DownloadFiles:    user.DownloadFiles,
		DownloadFileSize: user.DownloadFileSize,
	}
}

// handleAPIUser returns one user's stats, ban state, and the leaderboard.
func handleAPIUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(r.URL.Query().Get("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "user_id 参数无效")
		return
	}

	user, found, err := db.GetUserStats(userID)
	if err != nil {
		logger.Warn("查询用户 %d 统计失败: %s", userID, err)
		writeJSONError(w, http.StatusInternalServerError, "查询用户统计失败")
		return
	}
	detail := userDetail{Found: found, IsOwner: config.OwnerChatID != 0 && userID == config.OwnerChatID}
	detail.UserID = userID
	if found {
		detail.userSummary = summarizeUser(user)
		detail.CreateTime = user.CreateTime
		detail.RecentDownloadTime = user.RecentDownloadTime
		detail.Language = user.UserLanguage
		if rank, err := db.UserDownloadRank(user.DownloadFiles); err == nil {
			detail.Rank = rank
		}
	}

	resp := userResponse{User: detail, Leaderboard: []userSummary{}}
	if ban, banned := db.GetBan(userID); banned {
		resp.Ban = &ban
	}
	top, err := db.TopUsersByDownloads(leaderboardSize, db.SortByFiles)
	if err != nil {
		logger.Warn("查询用户排行榜失败: %s", err)
	}
	for _, user := range top {
		resp.Leaderboard = append(resp.Leaderboard, summarizeUser(user))
	}
	writeJSON(w, resp)
}

type leaderboardResponse struct {
	Users   []userSummary `json:"users"`
	Total   int64         `json:"total"`
	Sort    string        `json:"sort"`
	Query   string        `json:"query,omitempty"`
	Matched int64         `json:"matched"`
}

// maxSearchQueryLen keeps a pathological query from reaching the database.
const maxSearchQueryLen = 64

// handleAPILeaderboard serves the standalone ranking view, optionally
// filtered by a fuzzy user search.
func handleAPILeaderboard(w http.ResponseWriter, r *http.Request) {
	limit := leaderboardSize
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeJSONError(w, http.StatusBadRequest, "limit 参数无效")
			return
		}
		limit = min(parsed, 200)
	}
	sort := db.ParseUserSort(r.URL.Query().Get("sort"))
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) > maxSearchQueryLen {
		writeJSONError(w, http.StatusBadRequest, "搜索关键词过长")
		return
	}

	users, err := db.SearchUsers(query, limit, sort)
	if err != nil {
		logger.Warn("查询用户排行榜失败: %s", err)
		writeJSONError(w, http.StatusInternalServerError, "查询排行榜失败")
		return
	}
	resp := leaderboardResponse{
		Users: make([]userSummary, 0, len(users)),
		Sort:  string(sort),
		Query: query,
	}
	for _, user := range users {
		resp.Users = append(resp.Users, summarizeUser(user))
	}
	if total, err := db.CountUsers(); err == nil {
		resp.Total = total
	}
	if matched, err := db.CountMatchingUsers(query); err == nil {
		resp.Matched = matched
	}
	writeJSON(w, resp)
}

type banActionRequest struct {
	UserID int64  `json:"user_id"`
	Reason string `json:"reason"`
	Silent bool   `json:"silent"`
}

func decodeBanAction(w http.ResponseWriter, r *http.Request) (banActionRequest, bool) {
	var req banActionRequest
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "仅支持 POST")
		return req, false
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil || req.UserID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "请求格式错误")
		return req, false
	}
	return req, true
}

// handleAPIUserBan bans (or silently bans) a user from the WebUI.
func handleAPIUserBan(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeBanAction(w, r)
	if !ok {
		return
	}
	if config.OwnerChatID != 0 && req.UserID == config.OwnerChatID {
		writeJSONError(w, http.StatusBadRequest, "不能封禁所有者")
		return
	}
	if err := db.BanUser(req.UserID, strings.TrimSpace(req.Reason), req.Silent); err != nil {
		logger.Error("WebUI 封禁用户 %d 失败: %s", req.UserID, err)
		writeJSONError(w, http.StatusInternalServerError, "封禁失败")
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// handleAPIUserUnban lifts a user's ban from the WebUI.
func handleAPIUserUnban(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeBanAction(w, r)
	if !ok {
		return
	}
	existed, err := db.UnbanUser(req.UserID)
	if err != nil {
		logger.Error("WebUI 解封用户 %d 失败: %s", req.UserID, err)
		writeJSONError(w, http.StatusInternalServerError, "解封失败")
		return
	}
	writeJSON(w, map[string]bool{"ok": true, "existed": existed})
}
