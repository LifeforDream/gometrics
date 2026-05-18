package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LifeforDream/gometrics/internal/repository"
	"github.com/LifeforDream/gometrics/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestUpdateMetric(t *testing.T) {
	type want struct {
		statusCode int
	}
	type inputParams struct {
		method      string
		metricType  string
		metricName  string
		metricValue string
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		input inputParams
		want  want
	}{
		{
			name: "valid gauge update",
			input: inputParams{
				method:      http.MethodPost,
				metricType:  "gauge",
				metricName:  "Alloc",
				metricValue: "23.5",
			},
			want: want{statusCode: http.StatusOK},
		},
		{
			name: "valid counter update",
			input: inputParams{
				method:      http.MethodPost,
				metricType:  "counter",
				metricName:  "pollCount",
				metricValue: "5",
			},
			want: want{statusCode: http.StatusOK},
		},
		{
			name: "invalid metric type",
			input: inputParams{
				method:      http.MethodPost,
				metricType:  "invalid",
				metricName:  "someMetric",
				metricValue: "22",
			},
			want: want{statusCode: http.StatusBadRequest},
		},
		{
			name: "invalid gauge value",
			input: inputParams{
				method:      http.MethodPost,
				metricType:  "gauge",
				metricName:  "Alloc",
				metricValue: "invalid",
			},
			want: want{statusCode: http.StatusBadRequest},
		},
		{
			name: "bad counter type",
			input: inputParams{
				method:      http.MethodPost,
				metricType:  "counter",
				metricName:  "pollCount",
				metricValue: "1.5",
			},
			want: want{statusCode: http.StatusBadRequest},
		},
		{
			name: "invalid method",
			input: inputParams{
				method:      http.MethodGet,
				metricType:  "counter",
				metricName:  "pollCount",
				metricValue: "1.5",
			},
			want: want{statusCode: http.StatusMethodNotAllowed},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := service.NewMetricService(&repository.MemStorage{})
			path := fmt.Sprintf("/update/%s/%s/%s", tt.input.metricType, tt.input.metricName, tt.input.metricValue)
			r := httptest.NewRequest(tt.input.method, path, nil)
			r.SetPathValue("type", tt.input.metricType)
			r.SetPathValue("name", tt.input.metricName)
			r.SetPathValue("value", tt.input.metricValue)
			w := httptest.NewRecorder()
			h := NewHandler(service)
			h.UpdateMetric(w, r)
			resp := w.Result()
			defer resp.Body.Close()
			io.Copy(io.Discard, resp.Body)

			assert.Equal(t, tt.want.statusCode, resp.StatusCode, "unexpected status code: got %d, want %d", resp.StatusCode, tt.want.statusCode)
		})
	}
}
