package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOwnerChatIDFromYAML(t *testing.T) {
	path := writeTestConfig(t, "123456789")
	settings, usedPath, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if settings.OwnerChatID != 123456789 {
		t.Fatalf("owner_chat_id = %d", settings.OwnerChatID)
	}
	if !filepath.IsAbs(usedPath) {
		t.Fatalf("config path is not absolute: %s", usedPath)
	}
}

func TestOwnerChatIDIsNotReadFromEnvironment(t *testing.T) {
	t.Setenv("TELEGRAM_OWNER_CHAT_ID", "999")
	path := writeTestConfig(t, "42")
	settings, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if settings.OwnerChatID != 42 {
		t.Fatalf("owner_chat_id = %d", settings.OwnerChatID)
	}
}

func TestOwnerChatIDZeroDisablesNotifications(t *testing.T) {
	path := writeTestConfig(t, "0")
	settings, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if settings.OwnerChatID != 0 {
		t.Fatalf("owner_chat_id = %d", settings.OwnerChatID)
	}
}

func TestInvalidConcurrencyFailsValidation(t *testing.T) {
	path := writeTestConfig(t, "1")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(string(data) + "\ndownload:\n  max_concurrency: 0\n  max_retry: 3\n")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(path); err == nil {
		t.Fatal("expected validation error")
	}
}

func writeTestConfig(t *testing.T, owner string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := "telegram:\n" +
		"  token: fake-token\n" +
		"  http_token: fake-token\n" +
		"  owner_chat_id: " + owner + "\n" +
		"download:\n" +
		"  max_concurrency: 2\n" +
		"  max_retry: 1\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
