// Package apperrors содержит типизированные ошибки domain-слоя. Хендлеры
// делают на них type-assert/errors.Is, чтобы выбрать нужный HTTP-статус —
// новые виды ошибок стоит добавлять сюда, а не сравнивать текст ошибки строкой.
package apperrors

import (
	"errors"
	"fmt"
)

var (
	// MetricNotFound возвращается репозиторием, когда метрика с запрошенным
	// именем отсутствует в хранилище.
	MetricNotFound = errors.New("metric not found")
	// NonexistentMetricType возвращается, когда MType метрики не равен
	// ни models.Counter, ни models.Gauge.
	NonexistentMetricType = errors.New("non-existent metric type")
	// EmptyCounterDelta возвращается ValidateMetric, когда для метрики типа
	// counter не задано поле Delta.
	EmptyCounterDelta = errors.New("empty delta field for counter")
	// EmptyGaugeValue возвращается ValidateMetric, когда для метрики типа
	// gauge не задано поле Value.
	EmptyGaugeValue = errors.New("empty value field for gauge")
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
