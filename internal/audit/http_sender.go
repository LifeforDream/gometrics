package audit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
)

// HTTPAuditSender — реализация Sender, отправляющая каждое событие аудита
// отдельным HTTP POST-запросом с JSON-телом по адресу address.
type HTTPAuditSender struct {
	address string
	logger  *zap.Logger
	c       chan Event
}

// NewHTTPAuditSender создаёт HTTPAuditSender для указанного адреса addr,
// провалидировав его как URL.
func NewHTTPAuditSender(addr string, logger *zap.Logger) (*HTTPAuditSender, error) {
	_, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	return &HTTPAuditSender{address: addr, logger: logger}, nil
}

func (hs *HTTPAuditSender) setChan(c chan Event) {
	hs.c = c
}

func (hs *HTTPAuditSender) getID() string {
	return "http-audit-sender"
}

func (hs *HTTPAuditSender) worker() {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.Backoff = func(_, _ time.Duration, attemptNum int, _ *http.Response) time.Duration {
		return time.Duration(2*attemptNum+1) * time.Second
	}
	retryClient.Logger = nil
	retryClient.HTTPClient.Timeout = 3 * time.Second
	retryClient.ErrorHandler = retryablehttp.PassthroughErrorHandler
	client := retryClient.StandardClient()
	for event := range hs.c {
		hs.sendUpdate(event, client)
	}
}

func (hs *HTTPAuditSender) sendUpdate(ae Event, client *http.Client) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(ae); err != nil {
		hs.logger.Error("error encoding event for sending http audit event", zap.Error(err))
		return
	}
	req, err := http.NewRequest(http.MethodPost, hs.address, &buf)
	if err != nil {
		hs.logger.Error("error creating request for sending http audit event", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		hs.logger.Error("error response after sending http audit event", zap.Error(err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		hs.logger.Error("error response code after sending http audit event", zap.Int("response code", resp.StatusCode))
	}
}
