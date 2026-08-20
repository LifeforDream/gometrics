package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewFileAuditSender_Errors(t *testing.T) {
	// A path inside a non-existent directory can't be opened with O_CREATE.
	_, err := NewFileAuditSender(filepath.Join(t.TempDir(), "missing-dir", "audit.log"), zap.NewNop())
	assert.Error(t, err)
}

func TestFileAuditSender_GetID(t *testing.T) {
	fpath := filepath.Join(t.TempDir(), "audit.log")
	fas, err := NewFileAuditSender(fpath, zap.NewNop())
	require.NoError(t, err)
	assert.Equal(t, "file-audit-sender", fas.getID())
}

func TestFileAuditSender_WritesEventsAsJSONLines(t *testing.T) {
	fpath := filepath.Join(t.TempDir(), "audit.log")
	fas, err := NewFileAuditSender(fpath, zap.NewNop())
	require.NoError(t, err)

	c := make(chan Event)
	fas.setChan(c)
	done := make(chan struct{})
	go func() {
		fas.worker()
		close(done)
	}()

	c <- Event{Ts: 1, Metrics: []string{"Alloc"}, IPAddress: "1.2.3.4"}
	c <- Event{Ts: 2, Metrics: []string{"PollCount"}, IPAddress: "5.6.7.8"}
	close(c)
	<-done // worker() closes the file on exit, so the file is fully flushed here

	data, err := os.ReadFile(fpath)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	require.Len(t, lines, 2)

	var e1, e2 Event
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &e1))
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &e2))
	assert.Equal(t, Event{Ts: 1, Metrics: []string{"Alloc"}, IPAddress: "1.2.3.4"}, e1)
	assert.Equal(t, Event{Ts: 2, Metrics: []string{"PollCount"}, IPAddress: "5.6.7.8"}, e2)
}

func TestFileAuditSender_AppendsAcrossSenders(t *testing.T) {
	fpath := filepath.Join(t.TempDir(), "audit.log")

	writeOne := func(e Event) {
		fas, err := NewFileAuditSender(fpath, zap.NewNop())
		require.NoError(t, err)
		c := make(chan Event)
		fas.setChan(c)
		done := make(chan struct{})
		go func() {
			fas.worker()
			close(done)
		}()
		c <- e
		close(c)
		<-done
	}

	writeOne(Event{Ts: 1, Metrics: []string{"Alloc"}, IPAddress: "1.2.3.4"})
	writeOne(Event{Ts: 2, Metrics: []string{"PollCount"}, IPAddress: "5.6.7.8"})

	data, err := os.ReadFile(fpath)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	require.Len(t, lines, 2, "second sender should append, not truncate")
}
