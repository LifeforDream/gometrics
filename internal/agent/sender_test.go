package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendMetric(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		metricType  string
		metricName  string
		metricValue float64
		wantPath    string
		wantErr     bool
	}{
		{
			name:        "success counter send",
			metricType:  "counter",
			metricName:  "pollCount",
			metricValue: 5,
			wantPath:    "/update/counter/pollCount/5",
			wantErr:     false,
		},
		{
			name:        "success gauge send",
			metricType:  "gauge",
			metricName:  "Alloc",
			metricValue: 23.5,
			wantPath:    "/update/gauge/Alloc/23.500000",
			wantErr:     false,
		},
		{
			name:        "unsupported metric type",
			metricType:  "invalid",
			metricName:  "someMetric",
			metricValue: 22,
			wantPath:    "",
			wantErr:     true,
		},
	}
	//prepare
	client := &http.Client{}
	pathCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "unexpected HTTP method")
		assert.Equal(t, "text/plain", r.Header.Get("Content-Type"), "unexpected Content-Type")
		pathCh <- r.URL.Path
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
				// check path
				assert.Equal(t, tt.wantPath, <-pathCh, "unexpected request path")
			}
		})
	}
}
