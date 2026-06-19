package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	models "github.com/LifeforDream/gometrics/internal/model"
	"github.com/LifeforDream/gometrics/internal/repository"
	"github.com/LifeforDream/gometrics/internal/service"
	"github.com/LifeforDream/gometrics/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
	logger := zap.NewNop()

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
			input: []models.Metrics{{ID: "pollcount", MType: "counter", Delta: utils.IntPtr(t, 1)}},
			want:  []metric{{"counter", "pollcount", float64(1)}},
		},
		{
			name:  "a gauge",
			input: []models.Metrics{{ID: "alloc", MType: "gauge", Value: utils.FloatPtr(t, 1.25)}},
			want:  []metric{{"gauge", "alloc", 1.25}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			r := chi.NewRouter()
			service := service.NewMetricService(repository.NewMemStorage())

			h := NewHandler(service, logger)
			r.Get("/", h.GetMetrics)
			ts := httptest.NewServer(r)
			defer ts.Close()

			for _, metric := range tt.input {
				switch metric.MType {
				case "counter":
					service.UpdateCounter(t.Context(), metric)
				case "gauge":
					service.UpdateGauge(t.Context(), metric)
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
	logger := zap.NewNop()
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
			service.UpdateCounterByName(t.Context(), "pollcount", 2)
			service.UpdateGaugeByName(t.Context(), "alloc", 1.25)

			h := NewHandler(service, logger)
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
	logger := zap.NewNop()
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

			h := NewHandler(service, logger)
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

type stubService struct {
	*service.MetricService
	wantErr error
}

func (s *stubService) GetMetric(ctx context.Context, metricType, name string) (models.Metrics, error) {
	return models.Metrics{}, s.wantErr
}

func (s *stubService) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	return s.wantErr
}

func TestGetMetricJson(t *testing.T) {
	logger := zap.NewNop()
	defaultservice := service.NewMetricService(repository.NewMemStorage())
	defaultservice.UpdateCounter(t.Context(), models.Metrics{ID: "pollcount", MType: "counter", Delta: utils.IntPtr(t, 2)})
	defaultservice.UpdateGauge(t.Context(), models.Metrics{ID: "alloc", MType: "gauge", Value: utils.FloatPtr(t, 1.25)})

	type want struct {
		statusCode int
		metric     models.Metrics
	}
	tests := []struct {
		name        string
		svc         MetricService
		contentType string
		rawBody     string
		want        want
	}{
		{
			name:        "valid counter request",
			svc:         defaultservice,
			contentType: "application/json",
			rawBody:     `{"id":"pollcount","type":"counter"}`,
			want:        want{statusCode: http.StatusOK, metric: models.Metrics{ID: "pollcount", MType: "counter", Delta: utils.IntPtr(t, 2)}},
		},
		{
			name:        "valid gauge request",
			svc:         defaultservice,
			contentType: "application/json",
			rawBody:     `{"id":"alloc","type":"gauge"}`,
			want:        want{statusCode: http.StatusOK, metric: models.Metrics{ID: "alloc", MType: "gauge", Value: utils.FloatPtr(t, 1.25)}},
		},
		{
			name:        "wrong content type",
			svc:         defaultservice,
			contentType: "text/plain",
			rawBody:     `{"id":"pollcount","type":"counter"}`,
			want:        want{statusCode: http.StatusBadRequest},
		},
		{
			name:        "no content type",
			svc:         defaultservice,
			contentType: "",
			rawBody:     `{"id":"pollcount","type":"counter"}`,
			want:        want{statusCode: http.StatusBadRequest},
		},
		{
			name:        "malformed json",
			svc:         defaultservice,
			contentType: "application/json",
			rawBody:     `not-json`,
			want:        want{statusCode: http.StatusBadRequest},
		},
		{
			name:        "empty id",
			svc:         defaultservice,
			contentType: "application/json",
			rawBody:     `{"id":"","type":"counter"}`,
			want:        want{statusCode: http.StatusBadRequest},
		},
		{
			name:        "empty type",
			svc:         defaultservice,
			contentType: "application/json",
			rawBody:     `{"id":"pollcount","type":""}`,
			want:        want{statusCode: http.StatusBadRequest},
		},
		{
			name:        "metric not found",
			svc:         defaultservice,
			contentType: "application/json",
			rawBody:     `{"id":"unknown","type":"counter"}`,
			want:        want{statusCode: http.StatusNotFound},
		},
		{
			name:        "wrong metric type for existing metric",
			svc:         defaultservice,
			contentType: "application/json",
			rawBody:     `{"id":"alloc","type":"counter"}`,
			want:        want{statusCode: http.StatusBadRequest},
		},
		{
			name:        "500 error on infra problems",
			svc:         &stubService{wantErr: errors.New("some db error")},
			contentType: "application/json",
			rawBody:     `{"id":"alloc","type":"counter"}`,
			want:        want{statusCode: http.StatusInternalServerError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()

			h := NewHandler(tt.svc, logger)
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
	logger := zap.NewNop()
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
			h := NewHandler(service, logger)
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

func TestUpdateMetricsBatch(t *testing.T) {
	logger := zap.NewNop()
	type inputParams struct {
		contentType string
		rawBody     string
	}
	type want struct {
		statusCode int
	}
	tests := []struct {
		name  string
		svc   MetricService
		input inputParams
		want  want
	}{
		{
			name: "valid batch with gauge and counter",
			input: inputParams{
				contentType: "application/json",
				rawBody:     `[{"id":"alloc","type":"gauge","value":1.25},{"id":"pollcount","type":"counter","delta":5}]`,
			},
			want: want{statusCode: http.StatusOK},
		},
		{
			name: "wrong content type",
			input: inputParams{
				contentType: "text/plain",
				rawBody:     `[{"id":"alloc","type":"gauge","value":1.25}]`,
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
				rawBody:     `[{"id":"pollcount","type":"counter"}]`,
			},
			want: want{statusCode: http.StatusBadRequest},
		},
		{
			name: "gauge without value",
			input: inputParams{
				contentType: "application/json",
				rawBody:     `[{"id":"alloc","type":"gauge"}]`,
			},
			want: want{statusCode: http.StatusBadRequest},
		},
		{
			name: "unknown metric type in batch",
			input: inputParams{
				contentType: "application/json",
				rawBody:     `[{"id":"bad","type":"invalid","value":1.0}]`,
			},
			want: want{statusCode: http.StatusBadRequest},
		},
		{
			name: "500 on service error",
			svc:  &stubService{MetricService: service.NewMetricService(repository.NewMemStorage()), wantErr: errors.New("db error")},
			input: inputParams{
				contentType: "application/json",
				rawBody:     `[{"id":"alloc","type":"gauge","value":1.25}]`,
			},
			want: want{statusCode: http.StatusInternalServerError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := tt.svc
			if svc == nil {
				svc = service.NewMetricService(repository.NewMemStorage())
			}
			r := chi.NewRouter()
			h := NewHandler(svc, logger)
			r.Post("/updates", h.UpdateMetrics)
			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, _ := testRequestJSON(t, ts, http.MethodPost, "/updates", tt.input.contentType, tt.input.rawBody)
			assert.Equal(t, tt.want.statusCode, resp.StatusCode)
		})
	}
}

func TestPing(t *testing.T) {
	logger := zap.NewNop()
	tests := []struct {
		name       string
		connErr    error
		statusCode int
	}{
		{
			name:       "success 200",
			connErr:    nil,
			statusCode: http.StatusOK,
		},
		{
			name:       "error 500",
			connErr:    errors.New("connection unavailable"),
			statusCode: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err)
			defer db.Close()

			mock.ExpectPing().WillReturnError(tt.connErr)

			repo := repository.NewDbStorage(db)
			svc := service.NewMetricService(repo)
			h := NewHandler(svc, logger)

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			w := httptest.NewRecorder()

			h.Ping(w, req)

			assert.Equal(t, tt.statusCode, w.Code)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
