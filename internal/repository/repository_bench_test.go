package repository

import (
	"context"
	"testing"

	models "github.com/LifeforDream/gometrics/internal/model"
	"github.com/LifeforDream/gometrics/internal/utils"
)

// benchMetricsBatch строит батч метрик того же размера, что типичный
// отчёт агента: рантайм gauge-метрики вперемешку со счётчиком PollCount.
func benchMetricsBatch(n int) []models.Metrics {
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

// BenchmarkMemStorageUpdate измеряет батчевую запись MemStorage.UpdateMetrics —
// один mu.Lock() на весь батч через неэкспортированные хелперы setGauge/updateCounter.
func BenchmarkMemStorageUpdate(b *testing.B) {
	repo := NewMemStorage()
	batch := benchMetricsBatch(30)
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		if err := repo.UpdateMetrics(ctx, batch); err != nil {
			b.Fatal(err)
		}
	}
}
