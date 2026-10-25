package saga

import (
	"context"
	"fmt"
	"reflect"
)

type Yielder interface {
	context.Context
	Set(key, value any)
	Get(key any) (any, bool)
}

type YieldContext struct {
	context.Context
	parent Yielder
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

func (c *YieldContext) Set(key, value any) {
	if c.values == nil {
		c.values = make(map[any]any)
	}
	c.values[key] = value
}

func (c *YieldContext) Get(key any) (any, bool) {
	if c.values != nil {
		if value, ok := c.values[key]; ok {
			return value, true
		}
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
	if ctx == nil {
		return zero, fmt.Errorf("yield value of type %v not found", reflect.TypeFor[O]())
	}

	key := reflect.TypeFor[O]()
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
