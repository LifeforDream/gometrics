// Package models содержит общую модель данных метрик, используемую
// хендлером, сервисом и всеми реализациями репозитория.
package models

// Counter и Gauge — единственные допустимые значения поля Metrics.MType.
const (
	Counter = "counter"
	Gauge   = "gauge"
)

// Metrics — плоская модель метрики без иерархической вложенности.
// Gauge хранит значение в Value, counter — в Delta; оба поля объявлены
// через указатели, чтобы отличать значение "0" от не заданного значения
// и не усложнять структуру дополнительной кодировкой присутствия поля.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}
