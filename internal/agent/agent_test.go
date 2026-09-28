package agent

import (
	"context"
	"net/http"
	"testing"
	"testing/synctest"

	"go.uber.org/zap"
)

// TestAgentRunWaitGracefulShutdown проверяет, что после отмены контекста
// Agent.Wait() гарантированно дожидается завершения обеих дочерних горутин
// (collect и send), не паникует и не оставляет висящих горутин.
func TestAgentRunWaitGracefulShutdown(t *testing.T) {
	synctest.Test(t, func(_ *testing.T) {
		logger := zap.NewNop()
		client := newFakeClient(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		ctx, cancel := context.WithCancel(context.Background())
		a := New(Config{
			PollInterval:       3600,
			ReportInterval:     3600,
			ServerAddr:         "http://fake.invalid",
			ConcurrentRequests: 2,
			Client:             client,
		})

		a.Run(ctx, logger)
		cancel()

		a.Wait()
	})
}

// TestAgentWaitBlocksUntilShutdownComplete проверяет, что Wait() реально
// синхронизируется с завершением горутин агента, а не возвращается раньше
// времени, пока collect/send ещё работают.
func TestAgentWaitBlocksUntilShutdownComplete(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		logger := zap.NewNop()
		client := newFakeClient(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		ctx, cancel := context.WithCancel(context.Background())
		a := New(Config{
			PollInterval:       3600,
			ReportInterval:     3600,
			ServerAddr:         "http://fake.invalid",
			ConcurrentRequests: 1,
			Client:             client,
		})
		a.Run(ctx, logger)

		waitReturned := make(chan struct{})
		go func() {
			defer close(waitReturned)
			a.Wait()
		}()

		select {
		case <-waitReturned:
			t.Fatal("Wait() returned before the context was even cancelled")
		default:
		}

		cancel()
		<-waitReturned
	})
}

// TestAgentEndToEndFlushesDataOnShutdown проверяет полный путь данных через
// реальные collect/send: собранные метрики доходят до сервера, а после
// отмены контекста Wait() завершается без паники.
func TestAgentEndToEndFlushesDataOnShutdown(t *testing.T) {
	synctest.Test(t, func(_ *testing.T) {
		logger := zap.NewNop()
		hit := make(chan struct{}, 8)
		client := newFakeClient(func(w http.ResponseWriter, _ *http.Request) {
			select {
			case hit <- struct{}{}:
			default:
			}
			w.WriteHeader(http.StatusOK)
		})

		ctx, cancel := context.WithCancel(context.Background())
		a := New(Config{
			PollInterval:       1,
			ReportInterval:     1,
			ServerAddr:         "http://fake.invalid",
			ConcurrentRequests: 2,
			Client:             client,
		})
		a.Run(ctx, logger)

		<-hit

		cancel()

		a.Wait()
	})
}
