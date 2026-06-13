package utils

import "testing"

func FloatPtr(t *testing.T, f float64) *float64 {
	t.Helper()
	return &f
}

func IntPtr(t *testing.T, i int64) *int64 {
	t.Helper()
	return &i
}
