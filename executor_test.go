package saga

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestRunSequence(t *testing.T) {
	var executed []string

	step := func(name string) *Node[struct{}, struct{}] {
		return Do(name, func(ctx context.Context, fn struct{}) (struct{}, error) {
			executed = append(executed, name)
			return struct{}{}, nil
		})
	}

	flow := Sequence(
		step("look item"),
		step("checkout item"),
		step("charge item"),
		step("ship item"),
	)

	if err := Run(context.Background(), flow, struct{}{}); err != nil {
		t.Fatal(err)
	}

	expected := []string{"look item", "checkout item", "charge item", "ship item"}

	if !reflect.DeepEqual(executed, expected) {
		t.Fatalf("got %v, want %v", executed, expected)
	}
}

func TestRunStopsOnFailure(t *testing.T) {
	var executed []string

	paymentErr := errors.New("payment failed")

	first := Do("reserve", func(context.Context, struct{}) (struct{}, error) {
		executed = append(executed, "reserve")
		return struct{}{}, nil
	})

	second := Do("charge", func(context.Context, struct{}) (struct{}, error) {
		executed = append(executed, "charge")
		return struct{}{}, paymentErr
	})

	third := Do("shipment", func(context.Context, struct{}) (struct{}, error) {
		executed = append(executed, "shipment")
		return struct{}{}, nil
	})

	err := Run(
		context.Background(),
		Sequence(first, second, third),
		struct{}{},
	)

	if !errors.Is(err, paymentErr) {
		t.Fatalf("expected payment error, got %v", err)
	}

	if !errors.Is(err, ErrorSequenceType) {
		t.Fatalf("expected sequence error, got %v", err)
	}

	expected := []string{"reserve", "charge"}

	if !reflect.DeepEqual(executed, expected) {
		t.Fatalf("got %v, want %v", executed, expected)
	}
}

func TestRunCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false

	flow := Do("reserve", func(
		context.Context, struct{},
	) (any, error) {
		called = true
		return nil, nil
	})

	err := Run(ctx, flow, struct{}{})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}

	if called {
		t.Fatal("step should not executed")
	}
}
