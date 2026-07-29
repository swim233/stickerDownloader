package db

import "testing"

func seedUsers(t *testing.T) {
	t.Helper()
	users := []UserData{
		{UserID: 1, FirstName: "Alpha", DownloadFiles: 50, DownloadFileSize: 5000},
		{UserID: 2, FirstName: "Beta", DownloadFiles: 200, DownloadFileSize: 9000},
		{UserID: 3, FirstName: "Gamma", DownloadFiles: 10, DownloadFileSize: 100},
	}
	for _, user := range users {
		if err := DB.Create(&user).Error; err != nil {
			t.Fatalf("seed user %d: %v", user.UserID, err)
		}
	}
}

func TestGetUserStats(t *testing.T) {
	setupTestDB(t)
	seedUsers(t)

	user, found, err := GetUserStats(2)
	if err != nil || !found {
		t.Fatalf("GetUserStats(2) = %v, %v", found, err)
	}
	if user.FirstName != "Beta" || user.DownloadFiles != 200 {
		t.Fatalf("user = %+v", user)
	}

	_, found, err = GetUserStats(999)
	if err != nil || found {
		t.Fatalf("GetUserStats(999) = %v, %v; want not found, nil error", found, err)
	}
}

func TestTopUsersByDownloads(t *testing.T) {
	setupTestDB(t)
	seedUsers(t)

	top, err := TopUsersByDownloads(2, SortByFiles)
	if err != nil {
		t.Fatalf("top: %v", err)
	}
	if len(top) != 2 || top[0].UserID != 2 || top[1].UserID != 1 {
		t.Fatalf("top = %+v, want users 2 then 1", top)
	}
}

func TestUserDownloadRank(t *testing.T) {
	setupTestDB(t)
	seedUsers(t)

	cases := map[int]int64{200: 1, 50: 2, 10: 3, 0: 4}
	for files, want := range cases {
		rank, err := UserDownloadRank(files)
		if err != nil {
			t.Fatalf("rank(%d): %v", files, err)
		}
		if rank != want {
			t.Fatalf("rank(%d) = %d, want %d", files, rank, want)
		}
	}
}

func TestTopUsersBySize(t *testing.T) {
	setupTestDB(t)
	// Beta has fewer files than Alpha but far more bytes, so the two
	// orderings must disagree.
	for _, user := range []UserData{
		{UserID: 1, FirstName: "Alpha", DownloadFiles: 100, DownloadFileSize: 1000},
		{UserID: 2, FirstName: "Beta", DownloadFiles: 10, DownloadFileSize: 999999},
	} {
		if err := DB.Create(&user).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	byFiles, err := TopUsersByDownloads(2, SortByFiles)
	if err != nil || byFiles[0].UserID != 1 {
		t.Fatalf("by files = %+v (%v), want Alpha first", byFiles, err)
	}
	bySize, err := TopUsersByDownloads(2, SortBySize)
	if err != nil || bySize[0].UserID != 2 {
		t.Fatalf("by size = %+v (%v), want Beta first", bySize, err)
	}
}

func TestParseUserSort(t *testing.T) {
	cases := map[string]UserSort{
		"size": SortBySize, "files": SortByFiles, "": SortByFiles, "nonsense": SortByFiles,
		"download_files; DROP TABLE user_data": SortByFiles,
	}
	for input, want := range cases {
		if got := ParseUserSort(input); got != want {
			t.Fatalf("ParseUserSort(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSearchUsers(t *testing.T) {
	setupTestDB(t)
	for _, user := range []UserData{
		{UserID: 1001, FirstName: "Alpha", LastName: "Wolf", UserName: "alphawolf", DownloadFiles: 10},
		{UserID: 1002, FirstName: "Beta", UserName: "beta_tester", DownloadFiles: 20},
		{UserID: 9999, FirstName: "小", LastName: "明", DownloadFiles: 30},
		{UserID: 1003, FirstName: "100%", UserName: "percent", DownloadFiles: 40},
	} {
		if err := DB.Create(&user).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	cases := []struct {
		name  string
		query string
		want  []int64
	}{
		{"partial first name", "alph", []int64{1001}},
		{"case insensitive", "ALPHA", []int64{1001}},
		{"partial username", "tester", []int64{1002}},
		{"by last name", "Wolf", []int64{1001}},
		{"full name across fields", "Alpha Wolf", []int64{1001}},
		{"cjk", "小", []int64{9999}},
		{"partial id", "999", []int64{9999}},
		{"no match", "zzzzz", nil},
		{"wildcards are literal", "100%", []int64{1003}},
		{"underscore is literal", "beta_tester", []int64{1002}},
		{"empty returns all", "", []int64{1003, 9999, 1002, 1001}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			users, err := SearchUsers(tc.query, 50, SortByFiles)
			if err != nil {
				t.Fatalf("search: %v", err)
			}
			got := make([]int64, len(users))
			for i, u := range users {
				got[i] = u.UserID
			}
			if len(got) != len(tc.want) {
				t.Fatalf("search(%q) = %v, want %v", tc.query, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("search(%q) = %v, want %v", tc.query, got, tc.want)
				}
			}

			matched, err := CountMatchingUsers(tc.query)
			if err != nil {
				t.Fatalf("count: %v", err)
			}
			if matched != int64(len(tc.want)) {
				t.Fatalf("CountMatchingUsers(%q) = %d, want %d", tc.query, matched, len(tc.want))
			}
		})
	}
}

func TestSearchUsersRespectsSortAndLimit(t *testing.T) {
	setupTestDB(t)
	for _, user := range []UserData{
		{UserID: 1, FirstName: "Test A", DownloadFiles: 5, DownloadFileSize: 5000},
		{UserID: 2, FirstName: "Test B", DownloadFiles: 50, DownloadFileSize: 100},
	} {
		if err := DB.Create(&user).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	bySize, _ := SearchUsers("Test", 10, SortBySize)
	if len(bySize) != 2 || bySize[0].UserID != 1 {
		t.Fatalf("by size = %+v, want user 1 first", bySize)
	}
	limited, _ := SearchUsers("Test", 1, SortByFiles)
	if len(limited) != 1 || limited[0].UserID != 2 {
		t.Fatalf("limited = %+v, want just user 2", limited)
	}
}
