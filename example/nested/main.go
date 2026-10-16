package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Aldiwildan77/saga/saga"
)

type Order struct {
	ID string
}

func main() {
	flow := saga.Sequence(
		saga.Do[Order]("validate", func(
			ctx context.Context, o Order,
		) (any, error) {
			fmt.Println("validate order")
			return nil, nil
		}),

		saga.Nested("fulfillment",
			saga.Do[Order]("reserve", func(
				ctx context.Context, o Order,
			) (any, error) {
				fmt.Println("reserve inventory")
				return nil, nil
			}),
			saga.Do[Order]("pack", func(
				ctx context.Context, o Order,
			) (any, error) {
				fmt.Println("pack order")
				return nil, nil
			}),
		),

		saga.Do[Order]("complete", func(
			ctx context.Context, o Order,
		) (any, error) {
			fmt.Println("complete order")
			return nil, nil
		}),
	)

	if err := saga.Run(context.Background(), flow, Order{ID: "order-100"}); err != nil {
		log.Fatal(err)
	}
}
