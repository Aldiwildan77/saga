package saga

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestParallelFailureCompensatesSuccessfulBranches(t *testing.T) {
	var compensated atomic.Int32

	paymentErr := errors.New("payment failed")

	reserve := Do("reserve", func(
		context.Context, struct{},
	) error {
		return nil
	}).Compensate(ParallelCompensations(
		func(context.Context, struct{}) error {
			compensated.Add(1)
			return nil
		},
	))

	charge := Do("charge", func(
		context.Context, struct{},
	) error {
		return paymentErr
	})

	err := Run(
		context.Background(),
		Parallel(reserve, charge),
		struct{}{},
	)

	if !errors.Is(err, paymentErr) {
		t.Fatalf("expected payment error, got %v", err)
	}

	if !errors.Is(err, ErrorParallelType) {
		t.Fatalf("expected parallel error, got %v", err)
	}

	if got := compensated.Load(); got != 1 {
		t.Fatalf("expected 1 compensation, got %d", got)
	}
}

func TestNestedFailureCompensatesCompletedSteps(t *testing.T) {
	var compensated atomic.Int32
	stepErr := errors.New("confirmation failed")

	reserve := Do("reserve", func(
		context.Context, struct{},
	) error {
		return nil
	}).Compensate(ParallelCompensations(
		func(context.Context, struct{}) error {
			compensated.Add(1)
			return nil
		},
	))

	confirm := Do("confirm", func(
		context.Context, struct{},
	) error {
		return stepErr
	})

	flow := Sequence(
		Nested("order", reserve, confirm),
	)

	err := Run(context.Background(), flow, struct{}{})

	if !errors.Is(err, stepErr) {
		t.Fatalf("expected confirmation error, got %v", err)
	}

	if got := compensated.Load(); got != 1 {
		t.Fatalf("expected 1 compensation, got %d", got)
	}
}
