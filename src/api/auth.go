package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/swim233/StickerDownloader/config"
)

const (
	sessionCookieName = "sd_session"
	sessionTTL        = 7 * 24 * time.Hour
	// loginFailureDelay slows down password brute-forcing.
	loginFailureDelay = 400 * time.Millisecond
)

var sessionStore = struct {
	mu     sync.Mutex
	tokens map[string]time.Time
}{tokens: map[string]time.Time{}}

func newSessionToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func createSession() (string, error) {
	token, err := newSessionToken()
	if err != nil {
		return "", err
	}
	now := time.Now()
	sessionStore.mu.Lock()
	defer sessionStore.mu.Unlock()
	for existing, expires := range sessionStore.tokens {
		if now.After(expires) {
			delete(sessionStore.tokens, existing)
		}
	}
	sessionStore.tokens[token] = now.Add(sessionTTL)
	return token, nil
}

func sessionValid(token string) bool {
	sessionStore.mu.Lock()
	defer sessionStore.mu.Unlock()
	expires, ok := sessionStore.tokens[token]
	if !ok {
		return false
	}
	if time.Now().After(expires) {
		delete(sessionStore.tokens, token)
		return false
	}
	return true
}

func authenticated(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookieName)
	return err == nil && cookie.Value != "" && sessionValid(cookie.Value)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// requireAuth guards WebUI data endpoints behind a valid login session.
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.WebUIPassword == "" {
			writeJSONError(w, http.StatusForbidden, "server.password 未配置，WebUI 已禁用")
			return
		}
		if !authenticated(r) {
			writeJSONError(w, http.StatusUnauthorized, "未登录")
			return
		}
		next(w, r)
	}
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "仅支持 POST")
		return
	}
	if config.WebUIPassword == "" {
		writeJSONError(w, http.StatusForbidden, "server.password 未配置，WebUI 已禁用")
		return
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if subtle.ConstantTimeCompare([]byte(body.Password), []byte(config.WebUIPassword)) != 1 {
		time.Sleep(loginFailureDelay)
		writeJSONError(w, http.StatusUnauthorized, "密码错误")
		return
	}

	token, err := createSession()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "创建会话失败")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, map[string]bool{"ok": true})
}
