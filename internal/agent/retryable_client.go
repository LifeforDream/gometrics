package agent

import (
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// newRetryableClient создает новый http.Client с поддержкой повторных попыток.
func newRetryableClient(retryMax, timeout int) *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = retryMax
	retryClient.Backoff = func(_, _ time.Duration, attemptNum int, _ *http.Response) time.Duration {
		return time.Duration(2*attemptNum+1) * time.Second
	}
	retryClient.HTTPClient.Timeout = time.Duration(timeout) * time.Second
	retryClient.Logger = nil
	return retryClient.StandardClient()
}
