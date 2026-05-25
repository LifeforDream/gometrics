package apperrors

import (
	"errors"
	"fmt"
)

var (
	MetricNotFound = errors.New("metric not found")
)

type InvalidMetricType struct {
	ExistingType string
	NewType      string
	MetricName   string
}

func (e InvalidMetricType) Error() string {
	return fmt.Sprintf("invalid metric type: trying to update %s with type %s, but it already exists with type %s", e.MetricName, e.NewType, e.ExistingType)
}
