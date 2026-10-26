package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Aldiwildan77/saga"
)

type Order struct {
	ID string
}

func main() {
	noop := func(name string) func(context.Context, Order) (Order, error) {
		return func(ctx context.Context, o Order) (Order, error) {
			fmt.Println(name, o.ID)
			return o, nil
		}
	}

	flow := saga.Sequence(
		saga.Do("validate-order", noop("validate")),

		saga.Nested("prepare-order",
			saga.Do("reserve-inventory", noop("reserve")).Compensate(
				saga.ParallelCompensations[Order](
					func(context.Context, Order) (any, error) { return nil, nil },
					func(context.Context, Order) (any, error) { return nil, nil },
				),
			),
			saga.Do("verify-address", noop("verify")),
		),

		saga.Parallel(
			saga.Do("charge-payment", noop("charge")).Compensate(
				saga.SequentialCompensations[Order](
					func(context.Context, Order) (any, error) { return nil, nil },
				),
			),
			saga.Do("create-shipment", noop("ship")),
		),
	)

	// Print the structure without running any step.
	tree, err := saga.Visualize(flow)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(tree)
}
