// Package apperrors содержит типизированные ошибки domain-слоя. Хендлеры
// делают на них type-assert/errors.Is, чтобы выбрать нужный HTTP-статус —
// новые виды ошибок стоит добавлять сюда, а не сравнивать текст ошибки строкой.
package apperrors

import (
	"errors"
	"fmt"
)

var (
	// ErrMetricNotFound возвращается репозиторием, когда метрика с запрошенным
	// именем отсутствует в хранилище.
	ErrMetricNotFound = errors.New("metric not found")
	// ErrNonexistentMetricType возвращается, когда MType метрики не равен
	// ни models.Counter, ни models.Gauge.
	ErrNonexistentMetricType = errors.New("non-existent metric type")
	// ErrEmptyCounterDelta возвращается ValidateMetric, когда для метрики типа
	// counter не задано поле Delta.
	ErrEmptyCounterDelta = errors.New("empty delta field for counter")
	// ErrEmptyGaugeValue возвращается ValidateMetric, когда для метрики типа
	// gauge не задано поле Value.
	ErrEmptyGaugeValue = errors.New("empty value field for gauge")
)

// InvalidMetricType возвращается при попытке обновить существующую метрику
// значением другого типа (например, counter вместо ранее сохранённого gauge).
type InvalidMetricType struct {
	ExistingType string
	NewType      string
	MetricName   string
}

func (e InvalidMetricType) Error() string {
	return fmt.Sprintf("invalid metric type: trying to update %s with type %s, but it already exists with type %s", e.MetricName, e.NewType, e.ExistingType)
}
