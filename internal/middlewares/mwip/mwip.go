// Package mwip содержит chi-мидлвары, работающие с IP-адресом клиента:
// определение адреса с сохранением в контекст запроса и проверку
// вхождения адреса в доверенную подсеть.
package mwip

import (
	"net"
	"net/http"
	"strings"

	"go.uber.org/zap"

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

// WithTrustedSubnet — мидлвар chi: пропускает только запросы, чей IP-адрес
// входит в доверенную подсеть network, остальным возвращает HTTP 403.
//
// Адрес берётся из заголовка X-Real-IP, а при его отсутствии — из RemoteAddr.
// Нераспознаваемый адрес считается недоверенным. Если network == nil
// (доверенная подсеть не задана), запросы пропускаются без проверки.
func WithTrustedSubnet(network *net.IPNet, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if network == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)
			if netIP := net.ParseIP(ip); netIP == nil || !network.Contains(netIP) {
				log.Warn("client request not from trusted network",
					zap.String("ip-address", ip),
					zap.String("network CIDR", network.String()),
				)
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
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
	return remoteHost(r)
}

// realIP возвращает адрес из X-Real-IP, а при его отсутствии — хост из RemoteAddr.
// X-Forwarded-For намеренно не учитывается.
func realIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	return remoteHost(r)
}

func remoteHost(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
