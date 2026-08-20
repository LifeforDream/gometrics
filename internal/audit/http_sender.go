package audit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

type HTTPAuditSender struct {
	address string
	logger  *zap.Logger
	c       chan Event
}

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
	client := &http.Client{
		Timeout: 3 * time.Second,
	}
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
