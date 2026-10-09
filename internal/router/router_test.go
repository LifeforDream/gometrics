package router

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/LifeforDream/gometrics/internal/audit"
	"github.com/LifeforDream/gometrics/internal/handler"
	"github.com/LifeforDream/gometrics/internal/middlewares/mwip"
	"github.com/LifeforDream/gometrics/internal/repository"
	"github.com/LifeforDream/gometrics/internal/service"
)

// TestWriteOnlyChainAppliedOnlyToUpdateRoutes проверяет, что мидлвары из
// writeOnly срабатывают только на маршрутах, изменяющих данные, а маршруты
// чтения их не проходят.
func TestWriteOnlyChainAppliedOnlyToUpdateRoutes(t *testing.T) {
	logger := zap.NewNop()
	repo := repository.NewMemStorage()
	svc := service.NewMetricService(repo, audit.NewAuditor(logger))
	h := handler.NewHandler(svc, logger)

	_, trusted, err := net.ParseCIDR("192.168.0.0/24")
	require.NoError(t, err)

	r := MetricsRouter(h,
		NewChain(chimiddleware.StripSlashes),
		NewChain(),
		NewChain(mwip.WithTrustedSubnet(trusted, logger)),
	)
	srv := httptest.NewServer(r)
	defer srv.Close()

	tests := []struct {
		name          string
		method        string
		path          string
		wantForbidden bool
	}{
		{name: "GET / is not restricted", method: http.MethodGet, path: "/"},
		{name: "GET /ping is not restricted", method: http.MethodGet, path: "/ping"},
		{name: "GET /value is not restricted", method: http.MethodGet, path: "/value/gauge/alloc"},
		{name: "POST /value is not restricted", method: http.MethodPost, path: "/value"},
		{name: "POST /update/{type}/{name}/{value} is restricted", method: http.MethodPost, path: "/update/gauge/alloc/1", wantForbidden: true},
		{name: "POST /update is restricted", method: http.MethodPost, path: "/update", wantForbidden: true},
		{name: "POST /updates is restricted", method: http.MethodPost, path: "/updates", wantForbidden: true},
		{name: "POST /updates/ with trailing slash is restricted", method: http.MethodPost, path: "/updates/", wantForbidden: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, srv.URL+tt.path, nil)
			require.NoError(t, err)
			req.Header.Set("X-Real-IP", "10.0.0.1") // вне доверенной подсети

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			if tt.wantForbidden {
				assert.Equal(t, http.StatusForbidden, resp.StatusCode)
			} else {
				assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
			}
		})
	}

	t.Run("trusted client passes writeOnly chain", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/update/gauge/alloc/1", nil)
		require.NoError(t, err)
		req.Header.Set("X-Real-IP", "192.168.0.10")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
