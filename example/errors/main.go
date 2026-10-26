package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Aldiwildan77/saga"
)

type Order struct {
	ID string
}

func main() {
	order := Order{ID: "order-300"}

	fmt.Println("1. nil root:")
	report(saga.Run[Order](context.Background(), nil, order))

	fmt.Println("2. task without execute function:")
	report(saga.Run(context.Background(), saga.Do[Order, Order]("missing", nil), order))

	fmt.Println("3. input type mismatch:")
	report(saga.Run(context.Background(), saga.Do[string, string]("typed", func(
		ctx context.Context, s string,
	) (string, error) {
		return s, nil
	}), order))

	fmt.Println("4. step failure inside a sequence:")
	report(saga.Run(context.Background(), saga.Sequence(
		saga.Do[Order, Order]("validate", func(ctx context.Context, o Order) (Order, error) {
			return o, errors.New("address is invalid")
		}),
	), order))

	fmt.Println("5. parallel branches failing:")
	report(saga.Run(context.Background(), saga.Parallel(
		saga.Do[Order, Order]("inventory", func(ctx context.Context, o Order) (Order, error) {
			return o, errors.New("inventory offline")
		}),
		saga.Do[Order, Order]("payment", func(ctx context.Context, o Order) (Order, error) {
			return o, errors.New("payment gateway timeout")
		}),
	), order))

	fmt.Println("6. compensation failure after a later step fails:")
	report(saga.Run(context.Background(), saga.Sequence(
		saga.Do[Order, Order]("reserve", func(ctx context.Context, o Order) (Order, error) {
			return o, nil
		}).Compensate(saga.SequentialCompensations[Order](
			func(context.Context, Order) (any, error) {
				return nil, errors.New("warehouse rejected release")
			},
		)),
		saga.Do[Order, Order]("charge", func(ctx context.Context, o Order) (Order, error) {
			return o, errors.New("card declined")
		}),
	), order))

	fmt.Println("7. cancelled context:")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report(saga.Run(ctx, saga.Do[Order, Order]("reserve", func(ctx context.Context, o Order) (Order, error) {
		return o, nil
	}), order))

	fmt.Println("8. yield value missing:")
	report(saga.Run(context.Background(), saga.Do[Order, Order]("ship", func(ctx context.Context, o Order) (Order, error) {
		y := ctx.(saga.Yielder)
		if _, err := saga.YieldValue[time.Duration](y); err != nil {
			return o, err
		}
		return o, nil
	}), order))
}

func report(err error) {
	if err == nil {
		fmt.Println("   ok")
		return
	}

	fmt.Println("   error:", err)

	sentinels := []struct {
		name   string
		target error
	}{
		{"ErrorRootEmpty", saga.ErrorRootEmpty},
		{"ErrorNoneKind", saga.ErrorNoneKind},
		{"ErrorNoneMode", saga.ErrorNoneMode},
		{"ErrorSequenceType", saga.ErrorSequenceType},
		{"ErrorParallelType", saga.ErrorParallelType},
		{"ErrorCompensation", saga.ErrorCompensation},
		{"context.Canceled", context.Canceled},
	}

	for _, s := range sentinels {
		if errors.Is(err, s.target) {
			fmt.Println("   matches:", s.name)
		}
	}
}
