package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/LifeforDream/gometrics/internal/audit"
	models "github.com/LifeforDream/gometrics/internal/model"
	"github.com/LifeforDream/gometrics/internal/repository"
	"github.com/LifeforDream/gometrics/internal/service"
	"github.com/LifeforDream/gometrics/internal/utils"
)

// benchBatch строит батч метрик того же размера, что типичный отчёт
// агента: рантайм-метрики (~27 gauge + PollCount) плюс метрики gopsutil.
func benchBatch(n int) []models.Metrics {
	batch := make([]models.Metrics, 0, n)
	for i := range n {
		if i%5 == 0 {
			batch = append(batch, models.Metrics{
				ID:    "PollCount",
				MType: models.Counter,
				Delta: utils.IntPtr(int64(i)),
			})
			continue
		}
		batch = append(batch, models.Metrics{
			ID:    "GaugeMetric",
			MType: models.Gauge,
			Value: utils.FloatPtr(float64(i) * 1.5),
		})
	}
	return batch
}

// BenchmarkHandlerUpdateMetrics измеряет полный путь POST /updates:
// чтение тела, JSON-декодирование, валидацию и запись в MemStorage.
func BenchmarkHandlerUpdateMetrics(b *testing.B) {
	svc := service.NewMetricService(repository.NewMemStorage(), &audit.Auditor{})
	h := NewHandler(svc, zap.NewNop())

	r := chi.NewRouter()
	r.Post("/updates", h.UpdateMetrics)
	ts := httptest.NewServer(r)
	defer ts.Close()

	body, err := json.Marshal(benchBatch(30))
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/updates", bytes.NewReader(body))
		if err != nil {
			b.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b.Fatalf("unexpected status: %d", resp.StatusCode)
		}
	}
}
