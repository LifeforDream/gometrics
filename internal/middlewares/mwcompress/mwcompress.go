package mwcompress

import (
	"net/http"
	"strings"

	"github.com/LifeforDream/gometrics/internal/compress"
	"go.uber.org/zap"
)

func Compress(log *zap.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cw := compress.NewCompressWriter(w, strings.Contains(r.Header.Get("Accept-Encoding"), "gzip"))
			defer cw.Close()

			if r.Header.Get("Content-Encoding") == "gzip" {
				cr, err := compress.NewCompressReader(r.Body)
				if err != nil {
					log.Error("Error while decompressing body", zap.Error(err))
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				r.Body = cr
				defer cr.Close()
			}

			h.ServeHTTP(cw, r)
		})
	}
}
