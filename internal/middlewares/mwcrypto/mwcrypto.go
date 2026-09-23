package mwcrypto

import (
	"bytes"
	"crypto/rsa"
	"errors"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/LifeforDream/gometrics/internal/crypto"
)

const requestBodyLimit = (1 << 20) * 10 // 10 мегабайт

// WithCrypto осуществляет декодирование запроса с использованием приватного ключа.
//
// Если key не задан (nil), запрос пропускает как есть.
func WithCrypto(key *rsa.PrivateKey, log *zap.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key != nil {
				r.Body = http.MaxBytesReader(w, r.Body, requestBodyLimit)
				bodyBytes, err := io.ReadAll(r.Body)
				if err != nil {
					if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
						log.Error("Rejected payload: exceeded limit", zap.Int64("limit", maxBytesErr.Limit))
						w.WriteHeader(http.StatusRequestEntityTooLarge)
						return
					}
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
				r.ContentLength = int64(len(decrypted))
			}

			h.ServeHTTP(w, r)
		})
	}
}
