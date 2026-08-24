// Package logs содержит chi-мидлвар, логирующий каждый обработанный запрос.
package logs

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

// WithLogging возвращает chi-мидлвар, логирующий URI, метод, длительность
// запроса, статус ответа и размер тела на уровне Debug — уровень выбран
// намеренно, чтобы не засорять вывод автотестов. Статус 0 (когда хендлер
// вызывает только Write без явного WriteHeader) нормализуется до 200.
func WithLogging(log *zap.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			uri := r.RequestURI
			method := r.Method

			responseData := &responseData{
				status: 0,
				size:   0,
			}
			lw := loggingResponseWriter{
				ResponseWriter: w,
				responseData:   responseData,
			}
			h.ServeHTTP(&lw, r)

			duration := time.Since(start)

			status := lw.responseData.status
			if status == 0 {
				status = http.StatusOK
			}

			log.Debug("Request processed: ",
				zap.String("uri", uri),
				zap.String("method", method),
				zap.Duration("duration", duration),
				zap.Int("status", status),
				zap.Int("size", lw.responseData.size),
			)

		})
	}
}
