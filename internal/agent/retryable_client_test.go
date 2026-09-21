package agent

import (
	"net/http"
	"testing"
	"testing/synctest"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/require"
)

func TestNewRetryableClient(t *testing.T) {
	client := newRetryableClient(3, 5)
	if client == nil {
		t.Error("newRetryableClient() returned nil")
	}
}

func TestRetryableClientRetryMax(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cnt := 0
		retries := 2
		timeout := 1
		client := newRetryableClient(retries, timeout)
		rt := client.Transport.(*retryablehttp.RoundTripper)
		// подменяем не просто транспорт, а именно RoundTripper внутри retryablehttp.RoundTripper, чтобы
		// можно было контролировать поведение сервера и считать количество запросов.
		rt.Client.HTTPClient.Transport = newFakeTransport(func(w http.ResponseWriter, _ *http.Request) {
			cnt++
			w.WriteHeader(http.StatusInternalServerError)
		})

		_, err := client.Get("/")
		require.Error(t, err)
		if cnt != retries+1 { // 1 initial request + 2 retries
			t.Errorf("expected 4 requests, got %d", cnt)
		}
	})
}
