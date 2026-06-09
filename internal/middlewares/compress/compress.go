package compress

import (
	"net/http"
	"strings"
)

func Compress(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		cw := newCompressWriter(w, strings.Contains(r.Header.Get("Accept-Encoding"), "gzip"))
		defer cw.Close()

		if r.Header.Get("Content-Encoding") == "gzip" {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		next.ServeHTTP(cw, r)
	}
	return http.HandlerFunc(fn)
}
