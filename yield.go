package saga

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

type Yielder interface {
	context.Context
	Set(key, value any)
	Get(key any) (any, bool)
}

type YieldContext struct {
	context.Context
	parent Yielder
	mu     sync.RWMutex
	values map[any]any
}

func NewYieldContext(ctx context.Context) Yielder {
	if ctx == nil {
		ctx = context.Background()
	}

	if parent, ok := ctx.(Yielder); ok {
		return &YieldContext{
			Context: ctx,
			parent:  parent,
			values:  make(map[any]any),
		}
	}

	return &YieldContext{
		Context: ctx,
		values:  make(map[any]any),
	}
}

// Set is safe for concurrent use: parallel branches share one context.
func (c *YieldContext) Set(key, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.values == nil {
		c.values = make(map[any]any)
	}
	c.values[key] = value
}

func (c *YieldContext) Get(key any) (any, bool) {
	c.mu.RLock()
	value, ok := c.values[key]
	c.mu.RUnlock()

	if ok {
		return value, true
	}

	if c.parent != nil {
		return c.parent.Get(key)
	}

	return nil, false
}

func Yield[I any](ctx Yielder, value I) {
	if ctx == nil {
		return
	}

	ctx.Set(reflect.TypeFor[I](), value)
}

func YieldValue[O any](ctx Yielder) (O, error) {
	var zero O
	key := reflect.TypeFor[O]()

	if ctx == nil {
		return zero, fmt.Errorf("yield value of type %v not found", key)
	}

	value, ok := ctx.Get(key)
	if !ok {
		return zero, fmt.Errorf("yield value of type %v not found", key)
	}

	result, ok := value.(O)
	if !ok {
		return zero, fmt.Errorf("yield value type mismatch for %v", key)
	}

	return result, nil
}
