package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleConfig = `# 顶部注释
telegram:
  # Bot Token 注释
  token: "abc"
  debug: false
  owner_chat_id: 123

server:
  enabled: true
  port: ":8070"   # 端口注释
  password: "old"

supervisor:
  shutdown_timeout: 10s
  restart:
    initial_delay: 1s
    multiplier: 2
`

func writeSample(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(sampleConfig), 0600); err != nil {
		t.Fatalf("write sample: %v", err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(data)
}

func TestUpdateFilePreservesCommentsAndOrder(t *testing.T) {
	path := writeSample(t)
	err := UpdateFile(path, map[string]string{
		"server.port":     "127.0.0.1:9000",
		"telegram.debug":  "true",
		"server.password": "new-secret",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	got := readFile(t, path)

	for _, want := range []string{
		"# 顶部注释",
		"  # Bot Token 注释",
		`  token: "abc"`,
		"  debug: true",
		`  port: "127.0.0.1:9000"   # 端口注释`,
		`  password: "new-secret"`,
		"    initial_delay: 1s",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, `":8070"`) {
		t.Fatalf("old port still present:\n%s", got)
	}
}

func TestUpdateFileNestedKey(t *testing.T) {
	path := writeSample(t)
	if err := UpdateFile(path, map[string]string{"supervisor.restart.multiplier": "3"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	got := readFile(t, path)
	if !strings.Contains(got, "    multiplier: 3") {
		t.Fatalf("nested key not updated:\n%s", got)
	}
	// A same-named key at another level must not be touched by accident.
	if !strings.Contains(got, "  shutdown_timeout: 10s") {
		t.Fatalf("sibling key changed:\n%s", got)
	}
}

func TestUpdateFileAppendsMissingKey(t *testing.T) {
	path := writeSample(t)
	if err := UpdateFile(path, map[string]string{"server.api_token": "tok"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	got := readFile(t, path)
	if !strings.Contains(got, `  api_token: "tok"`) {
		t.Fatalf("key not appended:\n%s", got)
	}
	// It must land inside the server section, before supervisor.
	if strings.Index(got, "api_token") > strings.Index(got, "supervisor:") {
		t.Fatalf("key appended outside its section:\n%s", got)
	}
}

func TestUpdateFileUnknownSection(t *testing.T) {
	path := writeSample(t)
	if err := UpdateFile(path, map[string]string{"nope.key": "x"}); err == nil {
		t.Fatal("expected error for unknown section")
	}
}

func TestUpdateFileRoundTripsThroughLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := WriteExampleConfig(path); err != nil {
		t.Fatalf("write example: %v", err)
	}
	updates := map[string]string{
		"telegram.token":                "real-token",
		"telegram.http_token":           "real-http-token",
		"server.history_size":           "25",
		"server.password":               "pw",
		"image.jpeg_quality":            "80",
		"supervisor.restart.max_delay":  "2m",
		"notification.request_timeout":  "12s",
		"download.max_concurrency":      "40",
		"supervisor.restart.multiplier": "1.5",
	}
	if err := UpdateFile(path, updates); err != nil {
		t.Fatalf("update: %v", err)
	}
	settings, _, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v\n%s", err, readFile(t, path))
	}
	if settings.DownloadHistorySize != 25 || settings.JPEGQuality != 80 || settings.MaxConcurrency != 40 {
		t.Fatalf("scalars not applied: %+v", settings)
	}
	if settings.Supervisor.Restart.MaxDelay.String() != "2m0s" || settings.Notification.RequestTimeout.String() != "12s" {
		t.Fatalf("durations not applied: %+v", settings.Supervisor.Restart.MaxDelay)
	}
	if settings.Supervisor.Restart.Multiplier != 1.5 {
		t.Fatalf("float not applied: %v", settings.Supervisor.Restart.Multiplier)
	}
}

func TestFormatYAMLValue(t *testing.T) {
	cases := map[string]string{
		"":            `""`,
		"true":        "true",
		"42":          "42",
		"1.5":         "1.5",
		"10s":         `"10s"`,
		":8070":       `":8070"`,
		`say "hi"`:    `"say \"hi\""`,
		"C:\\path":    `"C:\\path"`,
		"plain-value": `"plain-value"`,
	}
	for input, want := range cases {
		if got := formatYAMLValue(input); got != want {
			t.Fatalf("formatYAMLValue(%q) = %q, want %q", input, got, want)
		}
	}
}
