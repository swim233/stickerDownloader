package handler

import (
	"testing"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
)

func TestParseBanArgs(t *testing.T) {
	cases := []struct {
		name    string
		args    string
		userID  int64
		reason  string
		wantErr bool
	}{
		{name: "id only", args: "12345", userID: 12345, reason: ""},
		{name: "id with reason", args: "12345 spamming stickers", userID: 12345, reason: "spamming stickers"},
		{name: "extra spaces", args: "  12345   多个  词  ", userID: 12345, reason: "多个 词"},
		{name: "empty", args: "", wantErr: true},
		{name: "not a number", args: "abc reason", wantErr: true},
		{name: "negative id", args: "-5 reason", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			userID, reason, err := parseBanArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseBanArgs(%q) expected error", tc.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseBanArgs(%q): %v", tc.args, err)
			}
			if userID != tc.userID || reason != tc.reason {
				t.Fatalf("parseBanArgs(%q) = %d, %q; want %d, %q", tc.args, userID, reason, tc.userID, tc.reason)
			}
		})
	}
}

func TestUpdateUserID(t *testing.T) {
	msg := tgbotapi.Update{Message: &tgbotapi.Message{From: &tgbotapi.User{ID: 11}}}
	if got := updateUserID(msg); got != 11 {
		t.Fatalf("message user = %d, want 11", got)
	}
	callback := tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{From: &tgbotapi.User{ID: 22}}}
	if got := updateUserID(callback); got != 22 {
		t.Fatalf("callback user = %d, want 22", got)
	}
	if got := updateUserID(tgbotapi.Update{}); got != 0 {
		t.Fatalf("empty update user = %d, want 0", got)
	}
}
