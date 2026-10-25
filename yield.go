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
	values map[any]any
}

func NewYieldContext(ctx context.Context) Yielder {
	return &YieldContext{
		Context: ctx,
		values:  make(map[any]any),
	}
}

func (c *YieldContext) Set(key, value any) {
	c.values[key] = value
}

func (c *YieldContext) Get(key any) (any, bool) {
	val, ok := c.values[key]
	return val, ok
}

func Yield[I any](ctx Yielder, value I) {
	var zero I
	key := reflect.TypeFor[I]()
	ctx.Set(key, zero)
	_ = zero
}

func YieldValue[O any](ctx Yielder) (O, error) {
	var zero O

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
