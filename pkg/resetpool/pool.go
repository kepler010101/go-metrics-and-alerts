package resetpool

import "sync"

// Resettable describes types that can reset their state.
type Resettable interface {
	Reset()
}

// Pool stores reusable objects that implement Resettable.
type Pool[T Resettable] struct {
	newFn func() T
	pool  sync.Pool
}

// New creates a pool for values produced by newFn.
// If newFn is nil, zero values will be returned.
func New[T Resettable](newFn func() T) *Pool[T] {
	p := &Pool[T]{newFn: newFn}
	p.pool.New = func() any {
		if newFn != nil {
			return newFn()
		}
		var zero T
		return zero
	}
	return p
}

// Get returns a value from the pool or creates a new one.
func (p *Pool[T]) Get() T {
	var zero T
	if p == nil {
		return zero
	}

	if got := p.pool.Get(); got != nil {
		if val, ok := got.(T); ok {
			return val
		}
	}

	if p.newFn != nil {
		return p.newFn()
	}

	return zero
}

// Put resets the value and returns it into the pool.
func (p *Pool[T]) Put(value T) {
	if p == nil {
		return
	}

	value.Reset()
	p.pool.Put(value)
}
