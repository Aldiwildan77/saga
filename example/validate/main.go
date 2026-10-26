package main

import (
	"context"
	"fmt"

	"github.com/Aldiwildan77/saga"
)

type Order struct {
	ID string
}

func main() {
	// This flow has three problems. Validate reports all of them at once.
	flow := saga.Sequence(
		saga.Do[Order, Order]("reserve", nil),

		saga.Parallel(),

		saga.Nested("finalize",
			saga.Do("confirm", func(_ context.Context, o Order) (Order, error) {
				return o, nil
			}).Compensate(saga.SequentialCompensations[Order](nil)),
		),
	)

	if err := saga.Validate(flow); err != nil {
		fmt.Println("invalid workflow:")
		fmt.Println(err)
		return
	}

	fmt.Println("workflow is valid")
}
