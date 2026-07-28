package api

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/swim233/StickerDownloader/config"
)

func setBehindProxy(t *testing.T, behind bool) {
	t.Helper()
	old := config.BehindProxy
	config.BehindProxy = behind
	t.Cleanup(func() { config.BehindProxy = old })
}

func setAPIToken(t *testing.T, token string) {
	t.Helper()
	old := config.APIToken
	config.APIToken = token
	t.Cleanup(func() { config.APIToken = old })
}

func TestRequestIsHTTPS(t *testing.T) {
	plain := httptest.NewRequest(http.MethodGet, "/", nil)

	setBehindProxy(t, false)
	if requestIsHTTPS(plain) {
		t.Fatal("plain request must not count as HTTPS")
	}

	forwarded := httptest.NewRequest(http.MethodGet, "/", nil)
	forwarded.Header.Set("X-Forwarded-Proto", "https")
	if requestIsHTTPS(forwarded) {
		t.Fatal("X-Forwarded-Proto must be ignored when behind_proxy is off")
	}

	setBehindProxy(t, true)
	if !requestIsHTTPS(forwarded) {
		t.Fatal("trusted X-Forwarded-Proto: https must count as HTTPS")
	}
	chained := httptest.NewRequest(http.MethodGet, "/", nil)
	chained.Header.Set("X-Forwarded-Proto", "https, http")
	if !requestIsHTTPS(chained) {
		t.Fatal("first value of a chained X-Forwarded-Proto must be used")
	}
	if requestIsHTTPS(plain) {
		t.Fatal("missing X-Forwarded-Proto must not count as HTTPS")
	}

	setBehindProxy(t, false)
	direct := httptest.NewRequest(http.MethodGet, "/", nil)
	direct.TLS = &tls.ConnectionState{}
	if !requestIsHTTPS(direct) {
		t.Fatal("direct TLS connection must count as HTTPS")
	}
}

func TestLoginCookieSecureFlagFollowsScheme(t *testing.T) {
	setWebUIPassword(t, "secret")

	setBehindProxy(t, false)
	rec := doLogin(t, "secret")
	if cookies := rec.Result().Cookies(); len(cookies) != 1 || cookies[0].Secure {
		t.Fatalf("plain HTTP cookie must not be Secure: %v", cookies)
	}

	setBehindProxy(t, true)
	req := httptest.NewRequest(http.MethodPost, "/api/login", newLoginBody("secret"))
	req.Header.Set("X-Forwarded-Proto", "https")
	rec = httptest.NewRecorder()
	handleLogin(rec, req)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure {
		t.Fatalf("HTTPS cookie must be Secure: %v", cookies)
	}
	if !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie lost its other protections: %+v", cookies[0])
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	setBehindProxy(t, false)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("missing CSP")
	}
	for _, directive := range []string{"default-src 'none'", "img-src 'self' data:", "media-src 'self'", "script-src 'self'"} {
		if !strings.Contains(csp, directive) {
			t.Fatalf("CSP missing %q: %s", directive, csp)
		}
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("missing frame protection")
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("HSTS must not be sent over plain HTTP, got %q", got)
	}

	setBehindProxy(t, true)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Fatal("HSTS missing over HTTPS")
	}
}

func TestRequireAPIToken(t *testing.T) {
	var reached bool
	guarded := requireAPIToken(func(w http.ResponseWriter, r *http.Request) { reached = true })

	call := func(target string, header string) int {
		reached = false
		req := httptest.NewRequest(http.MethodGet, target, nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		rec := httptest.NewRecorder()
		guarded(rec, req)
		return rec.Code
	}

	setAPIToken(t, "")
	if code := call("/stickerpack?name=x", ""); code != http.StatusOK || !reached {
		t.Fatalf("unset token must leave the endpoint open, code = %d", code)
	}

	setAPIToken(t, "s3cret")
	if code := call("/stickerpack?name=x", ""); code != http.StatusUnauthorized || reached {
		t.Fatalf("missing token must be rejected, code = %d", code)
	}
	if code := call("/stickerpack?name=x&token=wrong", ""); code != http.StatusUnauthorized || reached {
		t.Fatalf("wrong token must be rejected, code = %d", code)
	}
	if code := call("/stickerpack?name=x&token=s3cret", ""); code != http.StatusOK || !reached {
		t.Fatalf("query token must be accepted, code = %d", code)
	}
	if code := call("/stickerpack?name=x", "Bearer s3cret"); code != http.StatusOK || !reached {
		t.Fatalf("bearer token must be accepted, code = %d", code)
	}
}

func TestListensPublicly(t *testing.T) {
	cases := map[string]bool{
		":8070":            true,
		"0.0.0.0:8070":     true,
		"127.0.0.1:8070":   false,
		"localhost:8070":   false,
		"[::1]:8070":       false,
		"192.168.1.5:8070": true,
	}
	for addr, want := range cases {
		if got := listensPublicly(addr); got != want {
			t.Fatalf("listensPublicly(%q) = %v, want %v", addr, got, want)
		}
	}
}
