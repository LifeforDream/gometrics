// Package mwip содержит chi-мидлвар, определяющий IP-адрес клиента и
// кладущий его в контекст запроса.
package mwip

import (
	"net"
	"net/http"
	"strings"

	"github.com/LifeforDream/gometrics/internal/utils"
)

// WithClientIP — мидлвар chi: определяет IP-адрес входящего запроса
// и кладёт его в контекст. Достаётся через utils.ClientIP(ctx).
//
// За RemoteAddr в production-развёртывании обычно скрывается обратный
// прокси (nginx, балансировщик), поэтому реальный адрес клиента сначала
// ищется в заголовках X-Real-IP и X-Forwarded-For, которые проставляет
// сам прокси, и только при их отсутствии используется RemoteAddr.
func WithClientIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := utils.WithClientIP(r.Context(), clientIP(r))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ip := strings.TrimSpace(strings.Split(xff, ",")[0]); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
