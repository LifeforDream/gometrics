package audit

import (
	"sync"
	"testing"
	"time"

	models "github.com/LifeforDream/gometrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// fakeSender is a test double implementing the unexported Sender interface
// (same package, so unexported methods are reachable). It forwards every
// event it receives on its worker channel to a buffered `got` channel that
// tests can assert against, and closes `got` once the worker channel is
// closed — mirroring how a real sender shuts down on Auditor.Close.
type fakeSender struct {
	id  string
	c   chan Event
	got chan Event
}

func newFakeSender(id string) *fakeSender {
	return &fakeSender{id: id, got: make(chan Event, 10)}
}

func (f *fakeSender) getID() string        { return f.id }
func (f *fakeSender) setChan(c chan Event) { f.c = c }
func (f *fakeSender) worker() {
	for e := range f.c {
		f.got <- e
	}
	close(f.got)
}

func recvEvent(t *testing.T, c chan Event) Event {
	t.Helper()
	select {
	case e := <-c:
		return e
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit event")
		return Event{}
	}
}

func TestAuditorUpdate_NoSubscribers(t *testing.T) {
	a := NewAuditor(zap.NewNop())
	assert.NotPanics(t, func() {
		a.Update([]models.Metrics{{ID: "Alloc"}}, "127.0.0.1")
	})
}

func TestAuditorUpdate_DispatchesToSubscriber(t *testing.T) {
	a := NewAuditor(zap.NewNop())
	sender := newFakeSender("fake")
	a.RegisterSub(sender)

	before := time.Now().Unix()
	a.Update([]models.Metrics{{ID: "Alloc"}, {ID: "PollCount"}}, "192.168.0.42")
	after := time.Now().Unix()

	got := recvEvent(t, sender.got)
	assert.Equal(t, []string{"Alloc", "PollCount"}, got.Metrics)
	assert.Equal(t, "192.168.0.42", got.IPAddress)
	assert.GreaterOrEqual(t, got.Ts, before)
	assert.LessOrEqual(t, got.Ts, after)
}

func TestAuditorUpdate_BroadcastsToAllSubscribers(t *testing.T) {
	a := NewAuditor(zap.NewNop())
	s1 := newFakeSender("one")
	s2 := newFakeSender("two")
	a.RegisterSub(s1)
	a.RegisterSub(s2)

	a.Update([]models.Metrics{{ID: "Alloc"}}, "10.0.0.1")

	e1 := recvEvent(t, s1.got)
	e2 := recvEvent(t, s2.got)
	assert.Equal(t, []string{"Alloc"}, e1.Metrics)
	assert.Equal(t, []string{"Alloc"}, e2.Metrics)
}

func TestAuditorRegisterSub_SameIDDoesNotReplace(t *testing.T) {
	a := NewAuditor(zap.NewNop())
	s1 := newFakeSender("dup")
	s2 := newFakeSender("dup")
	a.RegisterSub(s1)
	a.RegisterSub(s2)

	require.Len(t, a.subChan, 1)

	a.Update([]models.Metrics{{ID: "Alloc"}}, "10.0.0.1")

	// Only the first registration's channel is retained by the auditor,
	// so only the first sender should observe the event.
	got := recvEvent(t, s1.got)
	assert.Equal(t, []string{"Alloc"}, got.Metrics)

	select {
	case e, ok := <-s2.got:
		t.Fatalf("expected no event on replaced subscriber, got %+v (ok=%v)", e, ok)
	case <-time.After(100 * time.Millisecond):
	}
}

// slowFakeSender delays before it starts draining its channel, so that
// several concurrent Update calls are still trying to deliver when the test
// calls Close — the exact window in which a fire-and-forget dispatch design
// (each Update spawning its own goroutine that sends straight to the
// subscriber channel) can panic with "send on closed channel".
type slowFakeSender struct {
	id         string
	startDelay time.Duration
	c          chan Event
	got        chan Event
}

func newSlowFakeSender(id string, startDelay time.Duration) *slowFakeSender {
	return &slowFakeSender{id: id, startDelay: startDelay, got: make(chan Event, 64)}
}

func (f *slowFakeSender) getID() string        { return f.id }
func (f *slowFakeSender) setChan(c chan Event) { f.c = c }
func (f *slowFakeSender) worker() {
	time.Sleep(f.startDelay)
	for e := range f.c {
		f.got <- e
	}
	close(f.got)
}

func TestAuditorClose_WaitsForInFlightUpdates(t *testing.T) {
	a := NewAuditor(zap.NewNop())
	sender := newSlowFakeSender("slow", 200*time.Millisecond)
	a.RegisterSub(sender)

	const n = 20
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			a.Update([]models.Metrics{{ID: "Alloc"}}, "127.0.0.1")
		})
	}
	wg.Wait() // every Update call has returned to its caller, though delivery may still be in flight

	a.Close() // must not panic even though the slow subscriber hasn't read anything yet

	got := 0
	for range sender.got {
		got++
	}
	assert.Equal(t, n, got, "Close must not drop events that were fired before it was called")
}

// stuckFakeSender never drains its channel, so dispatch permanently blocks
// trying to forward the first event to it — the rest of input's buffer
// fills up and stays full, giving a deterministic way to exercise the
// drop-on-full path regardless of goroutine scheduling.
type stuckFakeSender struct {
	id string
	c  chan Event
}

func (f *stuckFakeSender) getID() string        { return f.id }
func (f *stuckFakeSender) setChan(c chan Event) { f.c = c }
func (f *stuckFakeSender) worker()              { select {} }

func TestAuditorUpdate_DropsAndLogsWhenBufferFull(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	a := NewAuditor(zap.New(core))
	a.RegisterSub(&stuckFakeSender{id: "stuck"})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range inputBufferSize + 10 {
			a.Update([]models.Metrics{{ID: "Alloc"}}, "127.0.0.1")
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Update blocked instead of dropping once the input buffer filled")
	}

	assert.NotZero(t, logs.Len(), "expected at least one dropped-event warning once the buffer filled")
}

func TestAuditorClose_ShutsDownWorkers(t *testing.T) {
	a := NewAuditor(zap.NewNop())
	sender := newFakeSender("fake")
	a.RegisterSub(sender)

	a.Close()

	select {
	case _, ok := <-sender.got:
		assert.False(t, ok, "expected sender's got channel to be closed after Auditor.Close")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for worker to shut down after Close")
	}
}
