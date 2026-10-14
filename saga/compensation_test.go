package saga

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestSequentialCompensation(t *testing.T) {
	var calls []string

	task := Do("reserve", func(
		context.Context, struct{},
	) error {
		return nil
	}).Compensate(SequentialCompensations(
		func(context.Context, struct{}) error {
			calls = append(calls, "release")
			return nil
		},
		func(context.Context, struct{}) error {
			calls = append(calls, "notify")
			return nil
		},
	))

	failure := errors.New("payment failed")

	err := Run(context.Background(), Sequence(
		task,
		Do("charge", func(
			context.Context, struct{},
		) error {
			return failure
		}),
	), struct{}{})

	if !errors.Is(err, failure) {
		t.Fatalf("expected payment error, got %v", err)
	}

	expected := []string{"notify", "release"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("got %v, want %v", calls, expected)
	}
}

func TestCompensationErrorPreservesExecutionError(t *testing.T) {
	executionErr := errors.New("payment failed")
	compensationErr := errors.New("release failed")

	task := Do("reserve", func(
		context.Context, struct{},
	) error {
		return nil
	}).Compensate(ParallelCompensations(
		func(context.Context, struct{}) error {
			return compensationErr
		},
	))

	err := Run(context.Background(), Sequence(
		task,
		Do("charge", func(
			context.Context, struct{},
		) error {
			return executionErr
		}),
	), struct{}{})

	if !errors.Is(err, executionErr) {
		t.Fatalf("execution error lost: %v", err)
	}

	if !errors.Is(err, compensationErr) {
		t.Fatalf("compensation error lost: %v", err)
	}
}
