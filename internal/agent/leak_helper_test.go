package agent

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// dumpGoroutineStacks возвращает текстовый дамп стеков всех горутин
// процесса.
func dumpGoroutineStacks() string {
	buf := make([]byte, 1<<20)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return string(buf[:n])
		}
		buf = make([]byte, 2*len(buf))
	}
}

// assertNoGoroutineFunc проверяет, что среди горутин процесса не осталось ни
// одной со стеком, содержащим funcSubstr (например, "agent.worker(") — то
// есть что она гарантированно завершилась после graceful shutdown.
func assertNoGoroutineFunc(t *testing.T, funcSubstr string) {
	t.Helper()
	assert.NotContains(t, dumpGoroutineStacks(), funcSubstr, "goroutine still running: %s", funcSubstr)
}
