package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/swim233/StickerDownloader/config"
)

func setWebUIPassword(t *testing.T, password string) {
	t.Helper()
	old := config.WebUIPassword
	config.WebUIPassword = password
	t.Cleanup(func() { config.WebUIPassword = old })
}

func doLogin(t *testing.T, password string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{"password":"`+password+`"}`))
	rec := httptest.NewRecorder()
	handleLogin(rec, req)
	return rec
}

func TestLoginRejectedWithoutConfiguredPassword(t *testing.T) {
	setWebUIPassword(t, "")
	if rec := doLogin(t, "anything"); rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	setWebUIPassword(t, "secret")
	if rec := doLogin(t, "wrong"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestLoginGetNotAllowed(t *testing.T) {
	setWebUIPassword(t, "secret")
	rec := httptest.NewRecorder()
	handleLogin(rec, httptest.NewRequest(http.MethodGet, "/api/login", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestLoginSessionGrantsAccess(t *testing.T) {
	setWebUIPassword(t, "secret")
	loginRec := doLogin(t, "secret")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200", loginRec.Code)
	}
	cookies := loginRec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookieName || cookies[0].Value == "" {
		t.Fatalf("cookies = %v, want one non-empty %s cookie", cookies, sessionCookieName)
	}
	if !cookies[0].HttpOnly {
		t.Fatal("session cookie must be HttpOnly")
	}

	guarded := requireAuth(handleAPIStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	req.AddCookie(cookies[0])
	rec := httptest.NewRecorder()
	guarded(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("authed status = %d, want 200", rec.Code)
	}

	rec = httptest.NewRecorder()
	guarded(rec, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthed status = %d, want 401", rec.Code)
	}
}

func TestRequireAuthDisabledWithoutPassword(t *testing.T) {
	setWebUIPassword(t, "")
	rec := httptest.NewRecorder()
	requireAuth(handleAPIStatus)(rec, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}
