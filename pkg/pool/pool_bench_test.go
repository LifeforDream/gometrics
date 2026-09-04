package pool

import (
	"sync/atomic"
	"testing"
)

type resettableStruct struct {
	i int
	a string
	s []string
	m map[string]string
}

func (rs *resettableStruct) Reset() {
	rs.i = 0
	rs.a = ""
	rs.s = rs.s[:0]
	clear(rs.m)
}

// BenchmarkPool замеряет производительность с sync.Pool
func BenchmarkPool(b *testing.B) {
	p := New(func() *resettableStruct { return &resettableStruct{1, "a", make([]string, 5), make(map[string]string)} })
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			x := p.Get()
			p.Put(x)
		}
	})
}

var sink atomic.Pointer[resettableStruct]

// BenchmarkNoPool замеряет производительность с наивной фабрикой.
// Переменная sink нужна, чтобы компилятор не переоптимизировал цикл.
func BenchmarkNoPool(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		var x *resettableStruct
		for pb.Next() {
			x = &resettableStruct{1, "a", make([]string, 5), make(map[string]string)}
			x.Reset()
		}
		sink.Store(x)
	})
}
