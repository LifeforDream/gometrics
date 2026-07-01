package mwhash

import (
	"bytes"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/LifeforDream/gometrics/internal/utils"
	"go.uber.org/zap"
)

type hashWriter struct {
	http.ResponseWriter
	key string
}

func newHashWriter(key string, w http.ResponseWriter) *hashWriter {
	return &hashWriter{ResponseWriter: w, key: key}
}

func (hw *hashWriter) Write(p []byte) (int, error) {
	if hw.key != "" {
		hash := utils.GenSHA256(p, hw.key)
		hw.Header().Add(utils.HashHeaderName, hex.EncodeToString(hash))
	}
	return hw.ResponseWriter.Write(p)
}

func WithHash(key string, log *zap.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hw := newHashWriter(key, w)
			hashHeader := r.Header.Get(utils.HashHeaderName)
			if key != "" && hashHeader != "" {
				bodyBytes, err := io.ReadAll(r.Body)
				if err != nil {
					log.Error("error while reading request body", zap.Error(err))
				}

				// refill for further usage
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				bodyHash := hex.EncodeToString(utils.GenSHA256(bodyBytes, key))
				if bodyHash != hashHeader {
					log.Debug("body hash not equal to header hash", zap.String("bodyHash", bodyHash), zap.String("headerHash", hashHeader))
					w.WriteHeader(http.StatusBadRequest)
					return
				}

			}

			h.ServeHTTP(hw, r)
		})
	}
}
