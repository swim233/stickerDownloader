package db

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// GetUserStats returns the stored stats for a user; found is false when the
// user has never interacted with the bot.
func GetUserStats(userID int64) (UserData, bool, error) {
	var user UserData
	err := DB.Where("user_id = ?", userID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return UserData{}, false, nil
	}
	if err != nil {
		return UserData{}, false, err
	}
	return user, true, nil
}

// UserSort selects the leaderboard ordering.
type UserSort string

const (
	SortByFiles UserSort = "files"
	SortBySize  UserSort = "size"
)

// orderClause maps a sort choice onto a fixed SQL fragment. Keeping the
// clauses as constants — never interpolating caller input — is what makes the
// sort parameter safe to accept from the network.
func (s UserSort) orderClause() string {
	if s == SortBySize {
		return "download_file_size DESC, download_files DESC, user_id ASC"
	}
	return "download_files DESC, download_file_size DESC, user_id ASC"
}

// ParseUserSort normalises a sort name, falling back to the file count.
func ParseUserSort(name string) UserSort {
	if UserSort(name) == SortBySize {
		return SortBySize
	}
	return SortByFiles
}

// TopUsersByDownloads returns up to limit users ranked by the chosen metric.
func TopUsersByDownloads(limit int, sort UserSort) ([]UserData, error) {
	var users []UserData
	err := DB.Order(sort.orderClause()).Limit(limit).Find(&users).Error
	return users, err
}

// SearchUsers finds users whose name, username, or ID contains the query.
func SearchUsers(query string, limit int, sort UserSort) ([]UserData, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return TopUsersByDownloads(limit, sort)
	}
	// LIKE with escaped wildcards: a user searching for "100%" must not get
	// every row back.
	pattern := "%" + escapeLike(query) + "%"
	var users []UserData
	err := DB.Where(
		`first_name LIKE ?1 ESCAPE '\' OR last_name LIKE ?1 ESCAPE '\'
		 OR user_name LIKE ?1 ESCAPE '\' OR CAST(user_id AS TEXT) LIKE ?1 ESCAPE '\'
		 OR (first_name || ' ' || last_name) LIKE ?1 ESCAPE '\'`, pattern).
		Order(sort.orderClause()).Limit(limit).Find(&users).Error
	return users, err
}

// escapeLike neutralises the wildcards LIKE would otherwise interpret.
func escapeLike(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(s)
}

// CountMatchingUsers counts everyone a search would match, ignoring the limit.
func CountMatchingUsers(query string) (int64, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return CountUsers()
	}
	pattern := "%" + escapeLike(query) + "%"
	var total int64
	err := DB.Model(&UserData{}).Where(
		`first_name LIKE ?1 ESCAPE '\' OR last_name LIKE ?1 ESCAPE '\'
		 OR user_name LIKE ?1 ESCAPE '\' OR CAST(user_id AS TEXT) LIKE ?1 ESCAPE '\'
		 OR (first_name || ' ' || last_name) LIKE ?1 ESCAPE '\'`, pattern).Count(&total).Error
	return total, err
}

// CountUsers returns how many users have ever used the bot.
func CountUsers() (int64, error) {
	var total int64
	err := DB.Model(&UserData{}).Count(&total).Error
	return total, err
}

// UserDownloadRank returns the 1-based rank of a user by downloaded files.
func UserDownloadRank(downloadFiles int) (int64, error) {
	var ahead int64
	err := DB.Model(&UserData{}).Where("download_files > ?", downloadFiles).Count(&ahead).Error
	return ahead + 1, err
}
