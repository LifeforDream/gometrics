package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestHTTPAuditSender_GetID(t *testing.T) {
	hs, err := NewHTTPAuditSender("http://example.invalid", zap.NewNop())
	require.NoError(t, err)
	assert.Equal(t, "http-audit-sender", hs.getID())
}

func TestHTTPAuditSender_PostsEventAsJSON(t *testing.T) {
	received := make(chan Event, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var e Event
		require.NoError(t, json.NewDecoder(r.Body).Decode(&e))
		received <- e
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	hs, err := NewHTTPAuditSender(ts.URL, zap.NewNop())
	require.NoError(t, err)
	c := make(chan Event)
	hs.setChan(c)
	done := make(chan struct{})
	go func() {
		hs.worker()
		close(done)
	}()

	c <- Event{Ts: 1, Metrics: []string{"Alloc"}, IPAddress: "1.2.3.4"}

	select {
	case e := <-received:
		assert.Equal(t, Event{Ts: 1, Metrics: []string{"Alloc"}, IPAddress: "1.2.3.4"}, e)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server to receive audit event")
	}

	close(c)
	<-done
}

func TestHTTPAuditSender_LogsErrorResponseStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	core, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)

	hs, err := NewHTTPAuditSender(ts.URL, logger)
	require.NoError(t, err)
	c := make(chan Event)
	hs.setChan(c)
	done := make(chan struct{})
	go func() {
		hs.worker()
		close(done)
	}()

	c <- Event{Ts: 1, Metrics: []string{"Alloc"}, IPAddress: "1.2.3.4"}
	close(c)
	<-done

	require.Equal(t, 1, logs.Len())
	assert.Equal(t, 1, logs.FilterField(zap.Int("response code", http.StatusInternalServerError)).Len())
}

func TestHTTPAuditSender_LogsRequestError(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)

	// No server listening at this address: client.Do should fail outright.
	hs, err := NewHTTPAuditSender("http://127.0.0.1:0", logger)
	require.NoError(t, err)
	c := make(chan Event)
	hs.setChan(c)
	done := make(chan struct{})
	go func() {
		hs.worker()
		close(done)
	}()

	c <- Event{Ts: 1, Metrics: []string{"Alloc"}, IPAddress: "1.2.3.4"}
	close(c)
	<-done

	require.Equal(t, 1, logs.Len())
	assert.Contains(t, logs.All()[0].Message, "error response after sending http audit event")
}
