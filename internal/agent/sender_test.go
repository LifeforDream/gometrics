package agent

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	models "github.com/LifeforDream/gometrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var floatPtr = func(f float64) *float64 { return &f }
var intPtr = func(i int64) *int64 { return &i }

func TestSendMetric(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		metricType  string
		metricName  string
		metricValue float64
		wantMetric  models.Metrics
		wantErr     bool
	}{
		{
			name:        "success counter send",
			metricType:  "counter",
			metricName:  "pollCount",
			metricValue: 5,
			wantMetric:  models.Metrics{ID: "pollCount", MType: "counter", Delta: intPtr(5)},
			wantErr:     false,
		},
		{
			name:        "success gauge send",
			metricType:  "gauge",
			metricName:  "Alloc",
			metricValue: 23.5,
			wantMetric:  models.Metrics{ID: "Alloc", MType: "gauge", Value: floatPtr(23.5)},
			wantErr:     false,
		},
		{
			name:        "unsupported metric type",
			metricType:  "invalid",
			metricName:  "someMetric",
			metricValue: 22,
			wantErr:     true,
		},
	}
	client := &http.Client{}
	bodyCh := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "unexpected HTTP method")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "unexpected Content-Type")
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"), "unexpected Content-Encoding")
		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer gr.Close()
		body, err := io.ReadAll(gr)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		bodyCh <- body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := sendMetric(tt.metricType, tt.metricName, server.URL, tt.metricValue, client)
			if tt.wantErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
				var actMetric models.Metrics
				require.NoError(t, json.Unmarshal(<-bodyCh, &actMetric))
				assert.Equal(t, tt.wantMetric, actMetric, "wrong metric send result")
			}
		})
	}
}
