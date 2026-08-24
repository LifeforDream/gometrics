package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	models "github.com/LifeforDream/gometrics/internal/model"
	"github.com/LifeforDream/gometrics/internal/utils"
)

func TestSendMetricBatch(t *testing.T) {
	tests := []struct {
		name        string
		metrics     map[string]AgentMetric
		wantPayload []models.Metrics
		hashKey     string
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
				{ID: "alloc", MType: models.Gauge, Value: utils.FloatPtr(1.25)},
				{ID: "pollcount", MType: models.Counter, Delta: utils.IntPtr(3)},
			},
			hashKey:    "",
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
		{
			name: "setting hash key sets header",
			metrics: map[string]AgentMetric{
				"alloc":     {Type: models.Gauge, Value: 1.25},
				"pollcount": {Type: models.Counter, Value: 3},
			},
			wantPayload: []models.Metrics{
				{ID: "alloc", MType: models.Gauge, Value: utils.FloatPtr(1.25)},
				{ID: "pollcount", MType: models.Counter, Delta: utils.IntPtr(3)},
			},
			hashKey:    "somekey",
			hitsServer: true,
		},
	}
	client := &http.Client{}
	bodyCh := make(chan []byte, 1)
	hashHeaderCh := make(chan string, 1)
	rawBodyCh := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
		hashHeaderCh <- r.Header.Get(utils.HashHeaderName)
		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		rawBodyCh <- rawBody

		gr, err := gzip.NewReader(bytes.NewReader(rawBody))
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
			gotErr := sendMetricBatch(tt.metrics, server.URL, tt.hashKey, client)
			if tt.wantErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}
			if tt.hitsServer {
				var got []models.Metrics
				body := <-bodyCh
				require.NoError(t, json.Unmarshal(body, &got))
				assert.ElementsMatch(t, tt.wantPayload, got)

				hashHeader := <-hashHeaderCh
				rawBody := <-rawBodyCh
				if tt.hashKey != "" {
					assert.Equal(t, hex.EncodeToString(utils.GenSHA256(rawBody, tt.hashKey)), hashHeader)
				} else {
					assert.Empty(t, hashHeader)
				}
			}
		})
	}
}
