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
	notify := saga.Do[Order, Order]("notify", func(_ context.Context, o Order) (Order, error) {
		fmt.Println("notify", o.ID)
		return o, nil
	})

	// The same task appears in two places. That is reuse, not a cycle.
	flow := saga.Sequence(
		saga.Parallel(
			notify,
			saga.Do[Order, Order]("audit", func(_ context.Context, o Order) (Order, error) {
				return o, nil
			}),
		),
		saga.Nested("retry", notify),
	)

	if err := saga.DetectCycles(flow); err != nil {
		log.Fatal(err)
	}

	fmt.Println("no cycles found")
}
