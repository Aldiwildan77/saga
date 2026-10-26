package saga

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestYieldAndYieldValue(t *testing.T) {
	var got string

	flow := Sequence(
		Do("lookup", func(ctx context.Context, _ struct{}) (string, error) {
			y := ctx.(Yielder)
			Yield(y, "cust-42")
			return "cust-42", nil
		}),
		Do("ship", func(ctx context.Context, _ string) (struct{}, error) {
			value, err := YieldValue[string](ctx.(Yielder))
			if err != nil {
				return struct{}{}, err
			}

			got = value
			return struct{}{}, nil
		}),
	)

	if err := Run(context.Background(), flow, struct{}{}); err != nil {
		t.Fatal(err)
	}

	if got != "cust-42" {
		t.Fatalf("got %q, want %q", got, "cust-42")
	}
}

func TestYieldValueNotFound(t *testing.T) {
	var gotErr error

	flow := Do("ship", func(ctx context.Context, _ struct{}) (struct{}, error) {
		_, gotErr = YieldValue[string](ctx.(Yielder))
		return struct{}{}, nil
	})

	if err := Run(context.Background(), flow, struct{}{}); err != nil {
		t.Fatal(err)
	}

	if gotErr == nil {
		t.Fatal("expected not found error")
	}

	if !strings.Contains(gotErr.Error(), "not found") {
		t.Fatalf("unexpected error: %v", gotErr)
	}
}

func TestYieldValueNilYielder(t *testing.T) {
	value, err := YieldValue[string](nil)

	if err == nil {
		t.Fatal("expected error for nil yielder")
	}

	if value != "" {
		t.Fatalf("expected zero value, got %q", value)
	}
}

func TestYieldValueTypeMismatch(t *testing.T) {
	y := NewYieldContext(context.Background())

	// Bypass Yield to store a value whose dynamic type does not match the key.
	y.Set(reflect.TypeFor[int](), "not an int")

	value, err := YieldValue[int](y)

	if err == nil {
		t.Fatal("expected type mismatch error")
	}

	if !strings.Contains(err.Error(), "type mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}

	if value != 0 {
		t.Fatalf("expected zero value, got %d", value)
	}
}

func TestYieldNilYielderIsNoop(t *testing.T) {
	// Must not panic.
	Yield[string](nil, "ignored")
}

func TestYieldOverwritesSameType(t *testing.T) {
	y := NewYieldContext(context.Background())

	Yield(y, 1)
	Yield(y, 2)

	value, err := YieldValue[int](y)
	if err != nil {
		t.Fatal(err)
	}

	if value != 2 {
		t.Fatalf("got %d, want 2", value)
	}
}

func TestYieldFromNestedGroupVisibleToSibling(t *testing.T) {
	var got int

	flow := Sequence(
		Nested("inner",
			Do("produce", func(ctx context.Context, _ struct{}) (int, error) {
				Yield(ctx.(Yielder), 7)
				return 7, nil
			}),
		),
		Do("consume", func(ctx context.Context, _ int) (struct{}, error) {
			value, err := YieldValue[int](ctx.(Yielder))
			if err != nil {
				return struct{}{}, err
			}

			got = value
			return struct{}{}, nil
		}),
	)

	if err := Run(context.Background(), flow, struct{}{}); err != nil {
		t.Fatal(err)
	}

	if got != 7 {
		t.Fatalf("got %d, want 7", got)
	}
}

func TestYieldInParallelBranches(t *testing.T) {
	flow := Parallel(
		Do("int", func(ctx context.Context, _ struct{}) (int, error) {
			Yield(ctx.(Yielder), 1)
			return 1, nil
		}),
		Do("string", func(ctx context.Context, _ struct{}) (string, error) {
			Yield(ctx.(Yielder), "a")
			return "a", nil
		}),
		Do("bool", func(ctx context.Context, _ struct{}) (bool, error) {
			Yield(ctx.(Yielder), true)
			return true, nil
		}),
	)

	y := NewYieldContext(context.Background())
	if _, err := flow.run(y, struct{}{}); err != nil {
		t.Fatal(err)
	}

	if v, err := YieldValue[int](y); err != nil || v != 1 {
		t.Fatalf("int: got %d, err %v", v, err)
	}

	if v, err := YieldValue[string](y); err != nil || v != "a" {
		t.Fatalf("string: got %q, err %v", v, err)
	}

	if v, err := YieldValue[bool](y); err != nil || !v {
		t.Fatalf("bool: got %v, err %v", v, err)
	}
}

func TestRunWrapsContextAsYielder(t *testing.T) {
	flow := Do("check", func(ctx context.Context, _ struct{}) (struct{}, error) {
		if _, ok := ctx.(Yielder); !ok {
			return struct{}{}, errors.New("context is not a yielder")
		}

		return struct{}{}, nil
	})

	if err := Run(context.Background(), flow, struct{}{}); err != nil {
		t.Fatal(err)
	}
}

func TestRunNilContextFallsBackToBackground(t *testing.T) {
	var got bool

	flow := Do("check", func(ctx context.Context, _ struct{}) (struct{}, error) {
		got = ctx != nil && ctx.Err() == nil
		return struct{}{}, nil
	})

	//nolint:staticcheck // deliberately passing nil to verify the fallback
	if err := Run(nil, flow, struct{}{}); err != nil {
		t.Fatal(err)
	}

	if !got {
		t.Fatal("expected a usable background context")
	}
}

func TestNestedRunSeesParentYield(t *testing.T) {
	parent := NewYieldContext(context.Background())
	Yield(parent, "parent-value")

	var got string

	flow := Do("child", func(ctx context.Context, _ struct{}) (struct{}, error) {
		value, err := YieldValue[string](ctx.(Yielder))
		if err != nil {
			return struct{}{}, err
		}

		got = value
		return struct{}{}, nil
	})

	if err := Run(parent, flow, struct{}{}); err != nil {
		t.Fatal(err)
	}

	if got != "parent-value" {
		t.Fatalf("got %q, want %q", got, "parent-value")
	}
}

func TestChildYieldDoesNotLeakToParent(t *testing.T) {
	parent := NewYieldContext(context.Background())

	flow := Do("child", func(ctx context.Context, _ struct{}) (struct{}, error) {
		Yield(ctx.(Yielder), 99)
		return struct{}{}, nil
	})

	if err := Run(parent, flow, struct{}{}); err != nil {
		t.Fatal(err)
	}

	if _, ok := parent.Get(reflect.TypeFor[int]()); ok {
		t.Fatal("child yield should not leak into parent scope")
	}
}

func TestYieldContextGetFallsThroughToParent(t *testing.T) {
	parent := NewYieldContext(context.Background())
	parent.Set("key", "from-parent")

	child := NewYieldContext(parent)

	value, ok := child.Get("key")
	if !ok || value != "from-parent" {
		t.Fatalf("got %v (%v), want from-parent", value, ok)
	}

	if _, ok := child.Get("missing"); ok {
		t.Fatal("expected missing key to be absent")
	}
}

func TestYieldContextZeroValueSet(t *testing.T) {
	var c YieldContext
	c.Set("key", 1)

	if v, ok := c.Get("key"); !ok || v != 1 {
		t.Fatalf("got %v (%v), want 1", v, ok)
	}
}

func TestYieldValueErrorMessageNamesType(t *testing.T) {
	_, err := YieldValue[bool](NewYieldContext(context.Background()))

	want := fmt.Sprintf("yield value of type %v not found", reflect.TypeFor[bool]())
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
}
