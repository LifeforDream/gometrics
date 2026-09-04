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

func TestPut_ResetsObjectBeforeStoring(t *testing.T) {
	p := New(func() *resettableStub { return &resettableStub{} })
	obj := &resettableStub{Value: 100}

	p.Put(obj)

	assert.Equal(t, 0, obj.Value)
	assert.Equal(t, 1, obj.resetCalls)
}
