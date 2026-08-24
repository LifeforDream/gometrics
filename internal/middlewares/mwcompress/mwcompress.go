package mwcompress

import (
	"net/http"
	"slices"
	"strings"

	"go.uber.org/zap"

	"github.com/LifeforDream/gometrics/internal/compress"
)

var encodableTypes = []string{"application/json", "text/html"}

type zipWriter interface {
	Write([]byte) (int, error)
	Close() error
}

type compressWriter struct {
	w           http.ResponseWriter
	zw          zipWriter
	acceptsGzip bool
	wroteGzip   bool
}

func newCompressWriter(w http.ResponseWriter, zw zipWriter, acceptsGzip bool) *compressWriter {
	return &compressWriter{
		w:           w,
		zw:          zw,
		acceptsGzip: acceptsGzip,
	}
}

func (c *compressWriter) toEncode(w http.ResponseWriter) bool {
	return slices.Contains(encodableTypes, w.Header().Get("Content-Type")) && c.acceptsGzip
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {
	if c.toEncode(c.w) {
		c.wroteGzip = true
		// in case of missing WriteHeader() in handler
		c.w.Header().Set("Content-Encoding", "gzip")
		return c.zw.Write(p)
	}
	return c.w.Write(p)
}

func (c *compressWriter) WriteHeader(statusCode int) {
	// only works if w.Header().Set() was called before WriteHeader()
	if c.toEncode(c.w) && statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

func (c *compressWriter) Close() error {
	if c.wroteGzip {
		return c.zw.Close()
	}
	return nil
}

func Compress(log *zap.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cwr, err := compress.NewWriter(w)
			if err != nil {
				log.Error("error while creating compressed writer", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			cw := newCompressWriter(w, cwr, strings.Contains(r.Header.Get("Accept-Encoding"), "gzip"))
			defer cw.Close()

			if r.Header.Get("Content-Encoding") == "gzip" {
				cr, err := compress.NewCompressReader(r.Body)
				if err != nil {
					log.Error("error while decompressing body", zap.Error(err))
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
