// Package pool предоставляет структуру Pool и методы доступа к ней.
package pool

import "sync"

type resettable interface {
	Reset()
}

// Pool является структурой-контейнером для
// любых структур, имеющих метод Reset()
type Pool[T resettable] struct {
	pool sync.Pool
}

// New создаёт новый пустой Pool.
//
// Аргумент newF требует передать функцию-фабрику для новых объектов
func New[T resettable](newF func() T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				return newF()
			},
		},
	}
}

// Get переиспользует механизм sync.Pool.
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put помещает объект в пул, предварительно вызвав Reset.
func (p *Pool[T]) Put(x T) {
	x.Reset()
	p.pool.Put(x)
}
