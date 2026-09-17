package mwcrypto

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/LifeforDream/gometrics/internal/crypto"
)

// WithCrypto осуществляет декодирование запроса с использованием приватного ключа.
//
// Если key не задан (nil), запрос пропускает как есть.
func WithCrypto(key *rsa.PrivateKey, log *zap.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key != nil {
				bodyBytes, err := io.ReadAll(r.Body)
				if err != nil {
					log.Error("error while reading request body", zap.Error(err))
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				decrypted, err := crypto.Decrypt(key, bodyBytes)
				if err != nil {
					log.Warn("error while decrypting body with key configured", zap.Error(err))
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				// refill for further usage
				r.Body = io.NopCloser(bytes.NewBuffer(decrypted))
			}

			h.ServeHTTP(w, r)
		})
	}
}
