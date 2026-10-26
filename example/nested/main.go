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
	flow := saga.Sequence(
		saga.Do[Order, Order]("validate", func(
			ctx context.Context, o Order,
		) (Order, error) {
			fmt.Println("validate order")
			return o, nil
		}),

		saga.Nested("fulfillment",
			saga.Do[Order, Order]("reserve", func(
				ctx context.Context, o Order,
			) (Order, error) {
				fmt.Println("reserve inventory")
				return o, nil
			}),
			saga.Do[Order, Order]("pack", func(
				ctx context.Context, o Order,
			) (Order, error) {
				fmt.Println("pack order")
				return o, nil
			}),
		),

		saga.Do[Order, Order]("complete", func(
			ctx context.Context, o Order,
		) (Order, error) {
			fmt.Println("complete order")
			return o, nil
		}),
	)

	if err := saga.Run(context.Background(), flow, Order{ID: "order-100"}); err != nil {
		log.Fatal(err)
	}
}
