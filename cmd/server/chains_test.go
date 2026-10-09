package main

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBuildChains закрепляет состав и порядок цепочек, которые использует
// main: вместо настоящих мидлваров подставляются заглушки, записывающие своё
// имя при прохождении запроса. Поведение самих мидлваров и распределение
// маршрутов по группам проверяются в их пакетах и в internal/router.
func TestBuildChains(t *testing.T) {
	var calls []string
	rec := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, name)
				next.ServeHTTP(w, r)
			})
		}
	}

	// первый мидлвар в списке — внешний.
	run := func(chain mwList) []string {
		calls = nil
		var h http.Handler = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
		for _, c := range slices.Backward(chain) {
			h = c(h)
		}
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))
		return calls
	}

	core, read, write := buildChains(middlewares{
		stripSlashes:  rec("stripSlashes"),
		logging:       rec("logging"),
		clientIP:      rec("clientIP"),
		trustedSubnet: rec("trustedSubnet"),
		hash:          rec("hash"),
		crypto:        rec("crypto"),
		compress:      rec("compress"),
	})

	assert.Equal(t, []string{"stripSlashes", "logging", "clientIP"}, run(core), "core")
	assert.Equal(t, []string{"hash", "crypto", "compress"}, run(read), "read")
	assert.Equal(t, []string{"trustedSubnet", "hash", "crypto", "compress"}, run(write), "write")
}
