package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/LifeforDream/gometrics/internal/repository"
	"github.com/LifeforDream/gometrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRequest(t *testing.T, ts *httptest.Server, method, path string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestGetMetrics(t *testing.T) {
	type metric struct {
		mtype string
		name  string
		val   float64
	}
	tests := []struct {
		name  string
		input []metric //metrics to plant beforehand
		want  []metric //metrics to check
	}{
		{
			name: "no metrics",
		},
		{
			name:  "a counter",
			input: []metric{{"counter", "pollcount", float64(1)}},
			want:  []metric{{"counter", "pollcount", float64(1)}},
		},
		{
			name:  "a gauge",
			input: []metric{{"gauge", "alloc", 1.25}},
			want:  []metric{{"gauge", "alloc", 1.25}},
		},
	}
	r := chi.NewRouter()
	service := service.NewMetricService(&repository.MemStorage{})

	h := NewHandler(service)
	r.Get("/", h.GetMetrics)

	ts := httptest.NewServer(r)
	defer ts.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			for _, metric := range tt.input {
				switch metric.mtype {
				case "counter":
					service.UpdateCounter(metric.name, int64(metric.val))
				case "gauge":
					service.UpdateGauge(metric.name, metric.val)
				}
			}
			resp, get := testRequest(t, ts, http.MethodGet, "/")
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			for _, metric := range tt.want {
				assert.Contains(t, get, metric.name)
				assert.Contains(t, get, strconv.FormatFloat(metric.val, 'f', -1, 64))
			}
		})
	}
}

func TestGetMetric(t *testing.T) {
	type want struct {
		statusCode int
		value      string
	}
	type inputParams struct {
		metricType string
		metricName string
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		input inputParams
		want  want
	}{
		{
			name:  "valid request for counter",
			input: inputParams{"counter", "PollCount"},
			want:  want{200, "2"},
		},
		{
			name:  "case insensitive search",
			input: inputParams{"CouNter", "pollcount"},
			want:  want{200, "2"},
		},
		{
			name:  "valid request for gauge",
			input: inputParams{"gauge", "alloc"},
			want:  want{200, "1.25"},
		},
		{
			name:  "invalid metric type",
			input: inputParams{"counter", "alloc"},
			want:  want{400, ""},
		},
		{
			name:  "invalid metric name",
			input: inputParams{"counter", "a"},
			want:  want{404, ""},
		},
	}
	r := chi.NewRouter()
	service := service.NewMetricService(&repository.MemStorage{})
	service.UpdateCounter("pollcount", 2)
	service.UpdateGauge("alloc", 1.25)

	h := NewHandler(service)
	r.Get("/value/{type}/{name}", h.GetMetric)

	ts := httptest.NewServer(r)
	defer ts.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/value/%s/%s", tt.input.metricType, tt.input.metricName)
			resp, get := testRequest(t, ts, http.MethodGet, url)

			assert.Equal(t, tt.want.statusCode, resp.StatusCode)
			assert.Equal(t, tt.want.value, get)
		})
	}
}

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
			name: "type conflict: counter update on existing gauge",
			//only works after valid gauge update
			input: inputParams{
				method:      http.MethodPost,
				metricType:  "counter",
				metricName:  "Alloc",
				metricValue: "1",
			},
			want: want{statusCode: http.StatusBadRequest},
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
	r := chi.NewRouter()
	service := service.NewMetricService(&repository.MemStorage{})

	h := NewHandler(service)
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetric)

	ts := httptest.NewServer(r)
	defer ts.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("/update/%s/%s/%s", tt.input.metricType, tt.input.metricName, tt.input.metricValue)
			resp, _ := testRequest(t, ts, tt.input.method, path)
			assert.Equal(t, tt.want.statusCode, resp.StatusCode)
		})
	}
}
