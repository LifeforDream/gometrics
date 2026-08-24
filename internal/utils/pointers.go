package utils

// FloatPtr возвращает указатель на копию f — удобно для заполнения
// models.Metrics.Value литералом или результатом выражения.
func FloatPtr(f float64) *float64 {
	return &f
}

// IntPtr возвращает указатель на копию i — удобно для заполнения
// models.Metrics.Delta литералом или результатом выражения.
func IntPtr(i int64) *int64 {
	return &i
}
