package agent

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	models "github.com/LifeforDream/gometrics/internal/model"
	"github.com/LifeforDream/gometrics/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendMetricBatch(t *testing.T) {
	tests := []struct {
		name        string
		metrics     map[string]AgentMetric
		wantPayload []models.Metrics
		wantErr     bool
		hitsServer  bool
	}{
		{
			name: "success with gauge and counter",
			metrics: map[string]AgentMetric{
				"alloc":     {Type: models.Gauge, Value: 1.25},
				"pollcount": {Type: models.Counter, Value: 3},
			},
			wantPayload: []models.Metrics{
				{ID: "alloc", MType: models.Gauge, Value: utils.FloatPtr(t, 1.25)},
				{ID: "pollcount", MType: models.Counter, Delta: utils.IntPtr(t, 3)},
			},
			hitsServer: true,
		},
		{
			name:       "empty batch makes no HTTP call",
			metrics:    map[string]AgentMetric{},
			hitsServer: false,
		},
		{
			name:       "unknown metric type returns error",
			metrics:    map[string]AgentMetric{"bad": {Type: "invalid", Value: 1}},
			wantErr:    true,
			hitsServer: false,
		},
	}
	client := &http.Client{}
	bodyCh := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
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
			gotErr := sendMetricBatch(tt.metrics, server.URL, client)
			if tt.wantErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}
			if tt.hitsServer {
				var got []models.Metrics
				require.NoError(t, json.Unmarshal(<-bodyCh, &got))
				assert.ElementsMatch(t, tt.wantPayload, got)
			}
		})
	}
}
