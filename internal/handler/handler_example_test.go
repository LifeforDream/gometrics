package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/LifeforDream/gometrics/internal/audit"
	models "github.com/LifeforDream/gometrics/internal/model"
	"github.com/LifeforDream/gometrics/internal/repository"
	"github.com/LifeforDream/gometrics/internal/service"
	"github.com/LifeforDream/gometrics/internal/utils"
)

// exampleRequest issues an HTTP request without a body against a test
// server and returns the status code and response body. It panics on
// transport errors, which would indicate a bug in the example setup
// itself rather than in the handler under test.
func exampleRequest(ts *httptest.Server, method, path string) (int, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	if err != nil {
		panic(err)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return resp.StatusCode, strings.TrimSpace(string(body))
}

// exampleRequestJSON issues an HTTP request with a JSON body against a
// test server and returns the status code and response body.
func exampleRequestJSON(ts *httptest.Server, method, path, rawBody string) (int, string) {
	req, err := http.NewRequest(method, ts.URL+path, strings.NewReader(rawBody))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.Client().Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return resp.StatusCode, strings.TrimSpace(string(body))
}

// ExampleHandler_Ping demonstrates GET /ping, used to check that the
// server can reach its storage backend.
func ExampleHandler_Ping() {
	svc := service.NewMetricService(repository.NewMemStorage(), &audit.Auditor{})
	h := NewHandler(svc, zap.NewNop())

	r := chi.NewRouter()
	r.Get("/ping", h.Ping)
	ts := httptest.NewServer(r)
	defer ts.Close()

	status, _ := exampleRequest(ts, http.MethodGet, "/ping")
	fmt.Println(status)
	// Output:
	// 200
}

// ExampleHandler_GetMetrics demonstrates GET /, which renders an HTML
// page listing all known metrics.
func ExampleHandler_GetMetrics() {
	svc := service.NewMetricService(repository.NewMemStorage(), &audit.Auditor{})
	svc.UpdateGaugeByName(context.Background(), "Alloc", 1.25)

	h := NewHandler(svc, zap.NewNop())
	r := chi.NewRouter()
	r.Get("/", h.GetMetrics)
	ts := httptest.NewServer(r)
	defer ts.Close()

	status, body := exampleRequest(ts, http.MethodGet, "/")
	fmt.Println(status)
	fmt.Println(strings.Contains(body, "Alloc"))
	// Output:
	// 200
	// true
}

// ExampleHandler_GetMetricValue demonstrates GET /value/{type}/{name},
// which returns a single metric's value as plain text.
func ExampleHandler_GetMetricValue() {
	svc := service.NewMetricService(repository.NewMemStorage(), &audit.Auditor{})
	svc.UpdateCounterByName(context.Background(), "pollcount", 2)

	h := NewHandler(svc, zap.NewNop())
	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", h.GetMetricValue)
	ts := httptest.NewServer(r)
	defer ts.Close()

	status, body := exampleRequest(ts, http.MethodGet, "/value/counter/pollcount")
	fmt.Println(status)
	fmt.Println(body)
	// Output:
	// 200
	// 2
}

// ExampleHandler_GetMetric demonstrates POST /value, which returns a
// single metric encoded as JSON.
func ExampleHandler_GetMetric() {
	svc := service.NewMetricService(repository.NewMemStorage(), &audit.Auditor{})
	svc.UpdateGauge(context.Background(), models.Metrics{
		ID:    "Alloc",
		MType: models.Gauge,
		Value: utils.FloatPtr(1.25),
	})

	h := NewHandler(svc, zap.NewNop())
	r := chi.NewRouter()
	r.Post("/value", h.GetMetric)
	ts := httptest.NewServer(r)
	defer ts.Close()

	status, body := exampleRequestJSON(ts, http.MethodPost, "/value", `{"id":"Alloc","type":"gauge"}`)
	fmt.Println(status)
	fmt.Println(body)
	// Output:
	// 200
	// {"id":"Alloc","type":"gauge","value":1.25}
}

// ExampleHandler_UpdateMetricValue demonstrates
// POST /update/{type}/{name}/{value}, which updates a single metric via
// path parameters.
func ExampleHandler_UpdateMetricValue() {
	svc := service.NewMetricService(repository.NewMemStorage(), &audit.Auditor{})
	h := NewHandler(svc, zap.NewNop())

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetricValue)
	ts := httptest.NewServer(r)
	defer ts.Close()

	status, _ := exampleRequest(ts, http.MethodPost, "/update/gauge/Alloc/23.5")
	fmt.Println(status)
	// Output:
	// 200
}

// ExampleHandler_UpdateMetric demonstrates POST /update, which updates a
// single metric encoded as JSON.
func ExampleHandler_UpdateMetric() {
	svc := service.NewMetricService(repository.NewMemStorage(), &audit.Auditor{})
	h := NewHandler(svc, zap.NewNop())

	r := chi.NewRouter()
	r.Post("/update", h.UpdateMetric)
	ts := httptest.NewServer(r)
	defer ts.Close()

	status, _ := exampleRequestJSON(ts, http.MethodPost, "/update", `{"id":"PollCount","type":"counter","delta":5}`)
	fmt.Println(status)
	// Output:
	// 200
}

// ExampleHandler_UpdateMetrics demonstrates POST /updates, which updates
// a batch of metrics encoded as a JSON array in a single request.
func ExampleHandler_UpdateMetrics() {
	svc := service.NewMetricService(repository.NewMemStorage(), &audit.Auditor{})
	h := NewHandler(svc, zap.NewNop())

	r := chi.NewRouter()
	r.Post("/updates", h.UpdateMetrics)
	ts := httptest.NewServer(r)
	defer ts.Close()

	batch := `[{"id":"Alloc","type":"gauge","value":1.25},{"id":"PollCount","type":"counter","delta":5}]`
	status, _ := exampleRequestJSON(ts, http.MethodPost, "/updates", batch)
	fmt.Println(status)
	// Output:
	// 200
}
