package mwip

import (
	"net"
	"net/http"

	"github.com/LifeforDream/gometrics/internal/utils"
)

// WithClientIP — мидлвар chi: определяет IP-адрес входящего запроса
// и кладёт его в контекст. Достаётся через utils.ClientIP(ctx).
func WithClientIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := utils.WithClientIP(r.Context(), clientIP(r))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
