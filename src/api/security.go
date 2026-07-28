package api

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"

	"github.com/swim233/StickerDownloader/config"
	"github.com/swim233/StickerDownloader/logger"
)

// requestIsHTTPS reports whether the client's connection is encrypted, either
// directly or through a reverse proxy we were told to trust.
func requestIsHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if !config.BehindProxy {
		return false
	}
	// A proxy may chain values ("https, http"); the client-facing one is first.
	proto, _, _ := strings.Cut(r.Header.Get("X-Forwarded-Proto"), ",")
	return strings.EqualFold(strings.TrimSpace(proto), "https")
}

// securityHeaders applies hardening headers to every response. The WebUI is a
// single self-contained page, so a strict CSP costs nothing except allowing
// the page's own inline style and script blocks.
func securityHeaders(next http.Handler) http.Handler {
	const csp = "default-src 'none'; img-src 'self' data:; media-src 'self'; style-src 'self' 'unsafe-inline'; " +
		"script-src 'self' 'unsafe-inline'; connect-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Content-Security-Policy", csp)
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "no-referrer")
		if requestIsHTTPS(r) {
			header.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

// requestAPIToken pulls the token from an Authorization bearer header or the
// `token` query parameter.
func requestAPIToken(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); auth != "" {
		if token, ok := strings.CutPrefix(auth, "Bearer "); ok {
			return strings.TrimSpace(token)
		}
	}
	return r.URL.Query().Get("token")
}

// requireAPIToken guards the public download endpoint. With no token
// configured the endpoint stays open, matching its previous behaviour.
func requireAPIToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.APIToken == "" {
			next(w, r)
			return
		}
		if subtle.ConstantTimeCompare([]byte(requestAPIToken(r)), []byte(config.APIToken)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// listensPublicly reports whether the configured address accepts connections
// from outside the machine.
func listensPublicly(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = strings.TrimSuffix(addr, ":")
	}
	if host == "" {
		return true
	}
	if strings.EqualFold(host, "localhost") {
		return false
	}
	ip := net.ParseIP(host)
	return ip == nil || !ip.IsLoopback()
}

// logDeploymentWarnings surfaces risky combinations at startup.
func logDeploymentWarnings() {
	if config.WebUIPassword == "" {
		logger.Warn("server.password 未配置，WebUI 数据接口将拒绝访问")
	}
	if config.APIToken == "" {
		logger.Warn("server.api_token 未配置，/stickerpack 下载接口对所有访问者开放")
	}
	if listensPublicly(config.HTTPServerPort) && !config.BehindProxy {
		logger.Warn("HTTP 服务器监听 %s 且未启用 server.behind_proxy："+
			"若无 HTTPS 反向代理，密码与会话 Cookie 将以明文传输。"+
			"建议改为监听 127.0.0.1 并由反向代理提供 TLS", config.HTTPServerPort)
	}
}
