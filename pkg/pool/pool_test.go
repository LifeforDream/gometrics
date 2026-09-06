package pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resettableStub — тестовая структура с методом Reset(), удовлетворяющая
// generic-ограничению Pool. resetCalls считает вызовы Reset(), чтобы тесты
// могли проверить, что Put() действительно сбрасывает объект перед
// возвратом в пул.
type resettableStub struct {
	Value      int
	resetCalls int
}

func (r *resettableStub) Reset() {
	r.Value = 0
	r.resetCalls++
}

type (
	myInt32  int32
	myString string
)

func (mi myInt32) Reset() {
	mi = myInt32(0)
}

func (ms myString) Reset() {
	ms = ""
}

func TestNew_ReturnsNonNilPool(t *testing.T) {
	p := New(func() *resettableStub { return &resettableStub{} })

	require.NotNil(t, p)
}

func TestGet_ReturnsObjectFromFactoryWhenPoolEmpty(t *testing.T) {
	p := New(func() *resettableStub { return &resettableStub{Value: 42} })

	obj := p.Get()

	require.NotNil(t, obj)
	assert.Equal(t, 42, obj.Value)
}

func TestGet_DoesNotPanicOnNilFunc(t *testing.T) {
	tests := []struct {
		name string
		get  func() any
		want any
	}{
		{
			name: "test nil struct",
			get:  func() any { return New[*resettableStub](nil).Get() },
			want: (*resettableStub)(nil),
		},
		{
			name: "test nil int",
			get:  func() any { return New[myInt32](nil).Get() },
			want: myInt32(0),
		},
		{
			name: "test nil int",
			get:  func() any { return New[myString](nil).Get() },
			want: myString(""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.get())
		})
	}
}

func TestPut_ResetsObjectBeforeStoring(t *testing.T) {
	p := New(func() *resettableStub { return &resettableStub{} })
	obj := &resettableStub{Value: 100}

	p.Put(obj)

	assert.Equal(t, 0, obj.Value)
	assert.Equal(t, 1, obj.resetCalls)
}
