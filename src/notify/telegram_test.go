package notify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSendUsesOnlySendMessage(t *testing.T) {
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if !strings.HasSuffix(r.URL.Path, "/sendMessage") {
			t.Fatalf("unexpected endpoint: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "chat_id=42") {
			t.Fatalf("missing chat id: %s", body)
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer server.Close()

	n := New(Config{
		Token:       "fake-token",
		OwnerChatID: 42,
		APIEndpoint: server.URL + "/bot%s/%s",
		Timeout:     time.Second,
	})
	if err := n.Send(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestDisabledOwnerDoesNotSend(t *testing.T) {
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	}))
	defer server.Close()

	n := New(Config{Token: "fake-token", APIEndpoint: server.URL + "/bot%s/%s"})
	if err := n.Send(context.Background(), "ignored"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestSendEventDeduplicates(t *testing.T) {
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	n := New(Config{Token: "fake-token", OwnerChatID: 1, APIEndpoint: server.URL + "/bot%s/%s"})
	for range 2 {
		if err := n.SendEvent(context.Background(), "run:1:started", "started"); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestErrorDoesNotExposeToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"ok":false,"description":"upstream unavailable"}`))
	}))
	server.Close()

	const token = "super-secret-token"
	n := New(Config{Token: token, OwnerChatID: 1, APIEndpoint: server.URL + "/bot%s/%s", Timeout: 50 * time.Millisecond})
	err := n.Send(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("error exposed token: %v", err)
	}
}
