package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/swim233/StickerDownloader/config"
	"github.com/swim233/StickerDownloader/lib"
	"github.com/swim233/StickerDownloader/utils"
)

func TestSettingSpecsAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, spec := range settingSpecs {
		if seen[spec.Key] {
			t.Fatalf("duplicate setting key %s", spec.Key)
		}
		seen[spec.Key] = true
		if spec.current == nil {
			t.Fatalf("%s has no current() accessor", spec.Key)
		}
		if spec.Apply == applyHot && spec.hotApply == nil {
			t.Fatalf("%s claims hot apply but has no hotApply()", spec.Key)
		}
		if spec.Apply != applyHot && spec.hotApply != nil {
			t.Fatalf("%s has hotApply() but is not marked hot", spec.Key)
		}
		if spec.Danger && spec.Warning == "" {
			t.Fatalf("%s is dangerous but carries no warning", spec.Key)
		}
		if spec.Type == fieldSelect && len(spec.Options) == 0 {
			t.Fatalf("%s is a select with no options", spec.Key)
		}
		if !strings.Contains(spec.Key, ".") {
			t.Fatalf("%s is not a dotted config path", spec.Key)
		}
	}
}

func TestValidateSetting(t *testing.T) {
	specOf := func(key string) settingSpec {
		spec, ok := specByKey(key)
		if !ok {
			t.Fatalf("unknown spec %s", key)
		}
		return spec
	}
	cases := []struct {
		key     string
		value   string
		wantErr bool
	}{
		{"server.history_size", "50", false},
		{"server.history_size", "0", true},    // below min
		{"server.history_size", "5000", true}, // above max
		{"server.history_size", "abc", true},  // not an int
		{"image.jpeg_quality", "80", false},
		{"image.jpeg_quality", "101", true},
		{"telegram.debug", "true", false},
		{"telegram.debug", "yes", true},
		{"log.level", "DEBUG", false},
		{"log.level", "TRACE", true},
		{"notification.request_timeout", "12s", false},
		{"notification.request_timeout", "0s", true},
		{"notification.request_timeout", "soon", true},
		{"supervisor.restart.multiplier", "1.5", false},
		{"supervisor.restart.multiplier", "0.5", true},
		{"server.port", "127.0.0.1:9000", false},
		{"server.port", "bad\nvalue", true},
	}
	for _, tc := range cases {
		_, err := validateSetting(specOf(tc.key), tc.value)
		if tc.wantErr != (err != nil) {
			t.Fatalf("validateSetting(%s, %q) err = %v, wantErr %v", tc.key, tc.value, err, tc.wantErr)
		}
	}
}

func TestSecretHint(t *testing.T) {
	cases := map[string]string{"": "", "abc": "••••", "abcd": "••••", "supersecret": "••••cret"}
	for input, want := range cases {
		if got := secretHint(input); got != want {
			t.Fatalf("secretHint(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestHandleAPISettingsMasksSecrets(t *testing.T) {
	setWebUIPassword(t, "supersecret")

	rec := httptest.NewRecorder()
	handleAPISettings(rec, httptest.NewRequest(http.MethodGet, "/api/settings", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "supersecret") {
		t.Fatalf("raw secret leaked in response: %s", body)
	}
	var resp settingsResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, item := range resp.Settings {
		if item.Key == "server.password" {
			if item.Value != "••••cret" || !item.Set {
				t.Fatalf("password entry = %+v, want masked hint", item)
			}
			return
		}
	}
	t.Fatal("server.password missing from settings response")
}

// withTempConfig points the settings handlers at a throwaway config file.
func withTempConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := config.WriteExampleConfig(path); err != nil {
		t.Fatalf("write example config: %v", err)
	}
	// The example ships placeholder tokens that fail validation on reload.
	if err := config.UpdateFile(path, map[string]string{
		"telegram.token":      "test-token",
		"telegram.http_token": "test-http-token",
	}); err != nil {
		t.Fatalf("seed tokens: %v", err)
	}
	oldArgs := os.Args
	os.Args = []string{"worker", "--worker", "--config", path}
	t.Cleanup(func() { os.Args = oldArgs })
	return path
}

func TestHandleAPISettingsUpdateWritesAndHotApplies(t *testing.T) {
	path := withTempConfig(t)
	oldSize, oldQuality := config.DownloadHistorySize, config.JpegQuality
	oldCapacity := utils.DownloadHistory.Capacity()
	t.Cleanup(func() {
		config.DownloadHistorySize, config.JpegQuality = oldSize, oldQuality
		utils.DownloadHistory.SetCapacity(oldCapacity)
	})

	body := `{"updates":{"server.history_size":"42","image.jpeg_quality":"70","download.max_concurrency":"50"}}`
	rec := httptest.NewRecorder()
	handleAPISettingsUpdate(rec, httptest.NewRequest(http.MethodPost, "/api/settings/update", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}

	var resp settingsUpdateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Saved) != 3 {
		t.Fatalf("saved = %v, want 3 keys", resp.Saved)
	}
	if len(resp.AppliedNow) != 2 || len(resp.NeedsRestart) != 1 {
		t.Fatalf("apply split wrong: applied=%v restart=%v", resp.AppliedNow, resp.NeedsRestart)
	}
	if resp.NeedsRestart[0] != "download.max_concurrency" {
		t.Fatalf("needs_restart = %v", resp.NeedsRestart)
	}

	// Hot values applied in-process.
	if config.DownloadHistorySize != 42 || config.JpegQuality != 70 {
		t.Fatalf("hot apply failed: size=%d quality=%d", config.DownloadHistorySize, config.JpegQuality)
	}
	if utils.DownloadHistory.Capacity() != 42 {
		t.Fatalf("history capacity = %d, want 42", utils.DownloadHistory.Capacity())
	}

	// And persisted to disk so a restart keeps them.
	reloaded, _, err := config.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.DownloadHistorySize != 42 || reloaded.JPEGQuality != 70 || reloaded.MaxConcurrency != 50 {
		t.Fatalf("config file not updated: %+v", reloaded)
	}
}

func TestHandleAPISettingsUpdateRejectsBadInput(t *testing.T) {
	withTempConfig(t)
	cases := map[string]string{
		"unknown key":  `{"updates":{"nope.key":"1"}}`,
		"out of range": `{"updates":{"server.history_size":"0"}}`,
		"empty":        `{"updates":{}}`,
		"malformed":    `not json`,
	}
	for name, body := range cases {
		rec := httptest.NewRecorder()
		handleAPISettingsUpdate(rec, httptest.NewRequest(http.MethodPost, "/api/settings/update", strings.NewReader(body)))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400", name, rec.Code)
		}
	}
}

func TestHandleAPISettingsUpdateRejectsGet(t *testing.T) {
	rec := httptest.NewRecorder()
	handleAPISettingsUpdate(rec, httptest.NewRequest(http.MethodGet, "/api/settings/update", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestHandleAPIRestartSignalsOnce(t *testing.T) {
	// Drain any pending request so this test starts from a known state.
	select {
	case <-lib.RestartRequests():
	default:
	}

	rec := httptest.NewRecorder()
	handleAPIRestart(rec, httptest.NewRequest(http.MethodPost, "/api/restart", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("first restart status = %d, want 200", rec.Code)
	}

	rec = httptest.NewRecorder()
	handleAPIRestart(rec, httptest.NewRequest(http.MethodPost, "/api/restart", nil))
	if rec.Code != http.StatusConflict {
		t.Fatalf("second restart status = %d, want 409", rec.Code)
	}

	select {
	case reason := <-lib.RestartRequests():
		if reason == "" {
			t.Fatal("restart request carried no reason")
		}
	default:
		t.Fatal("no restart request queued")
	}
}
