package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	models "github.com/LifeforDream/gometrics/internal/model"
	"github.com/LifeforDream/gometrics/internal/repository"
	"github.com/LifeforDream/gometrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var floatPtr = func(f float64) *float64 { return &f }
var intPtr = func(i int64) *int64 { return &i }

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
		input []models.Metrics //metrics to plant beforehand
		want  []metric         //metrics to check
	}{
		{
			name: "no metrics",
		},
		{
			name:  "a counter",
			input: []models.Metrics{{ID: "pollcount", MType: "counter", Delta: intPtr(1)}},
			want:  []metric{{"counter", "pollcount", float64(1)}},
		},
		{
			name:  "a gauge",
			input: []models.Metrics{{ID: "alloc", MType: "gauge", Value: floatPtr(1.25)}},
			want:  []metric{{"gauge", "alloc", 1.25}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			r := chi.NewRouter()
			service := service.NewMetricService(repository.NewMemStorage())

			h := NewHandler(service)
			r.Get("/", h.GetMetrics)
			ts := httptest.NewServer(r)
			defer ts.Close()

			for _, metric := range tt.input {
				switch metric.MType {
				case "counter":
					service.UpdateCounter(metric)
				case "gauge":
					service.UpdateGauge(metric)
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			r := chi.NewRouter()
			service := service.NewMetricService(repository.NewMemStorage())
			service.UpdateCounterByName("pollcount", 2)
			service.UpdateGaugeByName("alloc", 1.25)

			h := NewHandler(service)
			r.Get("/value/{type}/{name}", h.GetMetricValue)

			ts := httptest.NewServer(r)
			defer ts.Close()

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
		setUp []inputParams
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
			setUp: []inputParams{
				{method: http.MethodPost, metricType: "gauge", metricName: "Alloc", metricValue: "23.5"},
			},
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//arrange
			r := chi.NewRouter()
			service := service.NewMetricService(repository.NewMemStorage())

			h := NewHandler(service)
			r.Post("/update/{type}/{name}/{value}", h.UpdateMetricValue)

			ts := httptest.NewServer(r)
			defer ts.Close()

			for _, s := range tt.setUp {
				path := fmt.Sprintf("/update/%s/%s/%s", s.metricType, s.metricName, s.metricValue)
				testRequest(t, ts, s.method, path)
			}

			path := fmt.Sprintf("/update/%s/%s/%s", tt.input.metricType, tt.input.metricName, tt.input.metricValue)
			resp, _ := testRequest(t, ts, tt.input.method, path)
			assert.Equal(t, tt.want.statusCode, resp.StatusCode)
		})
	}
}

func testRequestJSON(t *testing.T, ts *httptest.Server, method, path, contentType, rawBody string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, strings.NewReader(rawBody))
	require.NoError(t, err)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, string(respBody)
}

func TestGetMetricJson(t *testing.T) {
	type want struct {
		statusCode int
		metric     models.Metrics
	}
	tests := []struct {
		name        string
		contentType string
		rawBody     string
		want        want
	}{
		{
			name:        "valid counter request",
			contentType: "application/json",
			rawBody:     `{"id":"pollcount","type":"counter"}`,
			want:        want{statusCode: http.StatusOK, metric: models.Metrics{ID: "pollcount", MType: "counter", Delta: intPtr(2)}},
		},
		{
			name:        "valid gauge request",
			contentType: "application/json",
			rawBody:     `{"id":"alloc","type":"gauge"}`,
			want:        want{statusCode: http.StatusOK, metric: models.Metrics{ID: "alloc", MType: "gauge", Value: floatPtr(1.25)}},
		},
		{
			name:        "wrong content type",
			contentType: "text/plain",
			rawBody:     `{"id":"pollcount","type":"counter"}`,
			want:        want{statusCode: http.StatusBadRequest},
		},
		{
			name:        "no content type",
			contentType: "",
			rawBody:     `{"id":"pollcount","type":"counter"}`,
			want:        want{statusCode: http.StatusBadRequest},
		},
		{
			name:        "malformed json",
			contentType: "application/json",
			rawBody:     `not-json`,
			want:        want{statusCode: http.StatusBadRequest},
		},
		{
			name:        "empty id",
			contentType: "application/json",
			rawBody:     `{"id":"","type":"counter"}`,
			want:        want{statusCode: http.StatusBadRequest},
		},
		{
			name:        "empty type",
			contentType: "application/json",
			rawBody:     `{"id":"pollcount","type":""}`,
			want:        want{statusCode: http.StatusBadRequest},
		},
		{
			name:        "metric not found",
			contentType: "application/json",
			rawBody:     `{"id":"unknown","type":"counter"}`,
			want:        want{statusCode: http.StatusNotFound},
		},
		{
			name:        "wrong metric type for existing metric",
			contentType: "application/json",
			rawBody:     `{"id":"alloc","type":"counter"}`,
			want:        want{statusCode: http.StatusBadRequest},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			service := service.NewMetricService(repository.NewMemStorage())
			service.UpdateCounter(models.Metrics{ID: "pollcount", MType: "counter", Delta: intPtr(2)})
			service.UpdateGauge(models.Metrics{ID: "alloc", MType: "gauge", Value: floatPtr(1.25)})

			h := NewHandler(service)
			r.Post("/value", h.GetMetric)
			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, body := testRequestJSON(t, ts, http.MethodPost, "/value", tt.contentType, tt.rawBody)
			assert.Equal(t, tt.want.statusCode, resp.StatusCode)
			if tt.want.statusCode == http.StatusOK {
				var got models.Metrics
				require.NoError(t, json.NewDecoder(strings.NewReader(body)).Decode(&got))
				assert.Equal(t, tt.want.metric, got)
			}
		})
	}
}

func TestUpdateMetricJson(t *testing.T) {
	type inputParams struct {
		contentType string
		rawBody     string
	}
	type want struct {
		statusCode int
	}
	tests := []struct {
		name  string
		setUp []inputParams
		input inputParams
		want  want
	}{
		{
			name: "valid gauge update",
			input: inputParams{
				contentType: "application/json",
				rawBody:     `{"id":"alloc","type":"gauge","value":23.5}`,
			},
			want: want{statusCode: http.StatusOK},
		},
		{
			name: "valid counter update",
			input: inputParams{
				contentType: "application/json",
				rawBody:     `{"id":"pollcount","type":"counter","delta":5}`,
			},
			want: want{statusCode: http.StatusOK},
		},
		{
			name: "wrong content type",
			input: inputParams{
				contentType: "text/plain",
				rawBody:     `{"id":"alloc","type":"gauge","value":23.5}`,
			},
			want: want{statusCode: http.StatusBadRequest},
		},
		{
			name: "malformed json",
			input: inputParams{
				contentType: "application/json",
				rawBody:     `not-json`,
			},
			want: want{statusCode: http.StatusBadRequest},
		},
		{
			name: "counter without delta",
			input: inputParams{
				contentType: "application/json",
				rawBody:     `{"id":"pollcount","type":"counter"}`,
			},
			want: want{statusCode: http.StatusBadRequest},
		},
		{
			name: "gauge without value",
			input: inputParams{
				contentType: "application/json",
				rawBody:     `{"id":"alloc","type":"gauge"}`,
			},
			want: want{statusCode: http.StatusBadRequest},
		},
		{
			name: "invalid metric type",
			input: inputParams{
				contentType: "application/json",
				rawBody:     `{"id":"something","type":"invalid","value":1.0}`,
			},
			want: want{statusCode: http.StatusBadRequest},
		},
		{
			name: "type conflict: counter update on existing gauge",
			setUp: []inputParams{
				{contentType: "application/json", rawBody: `{"id":"alloc","type":"gauge","value":23.5}`},
			},
			input: inputParams{
				contentType: "application/json",
				rawBody:     `{"id":"alloc","type":"counter","delta":1}`,
			},
			want: want{statusCode: http.StatusBadRequest},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			service := service.NewMetricService(repository.NewMemStorage())
			h := NewHandler(service)
			r.Post("/update", h.UpdateMetric)
			ts := httptest.NewServer(r)
			defer ts.Close()

			for _, s := range tt.setUp {
				testRequestJSON(t, ts, http.MethodPost, "/update", s.contentType, s.rawBody)
			}

			resp, _ := testRequestJSON(t, ts, http.MethodPost, "/update", tt.input.contentType, tt.input.rawBody)
			assert.Equal(t, tt.want.statusCode, resp.StatusCode)
		})
	}
}
