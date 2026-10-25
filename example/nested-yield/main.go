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

type Customer struct {
	ID string
}

func main() {
	flow := saga.Sequence(
		saga.Nested("prepare-order",
			saga.Do[Order]("lookup-customer", func(ctx context.Context, order Order) (Customer, error) {
				y, ok := ctx.(saga.Yielder)
				if !ok {
					return Customer{}, fmt.Errorf("context is not yield-enabled")
				}

				customer := Customer{ID: "cust-42"}
				saga.Yield(y, customer)
				fmt.Println("customer resolved for order:", order.ID)
				return customer, nil
			}),
		),
		saga.Do[Order]("ship-order", func(ctx context.Context, order Order) (string, error) {
			y, ok := ctx.(saga.Yielder)
			if !ok {
				return "", fmt.Errorf("context is not yield-enabled")
			}

			customer, err := saga.YieldValue[Customer](y)
			if err != nil {
				return "", err
			}

			fmt.Printf("ship order %s to customer %s\n", order.ID, customer.ID)
			return customer.ID, nil
		}),
	)

	if err := saga.Run(context.Background(), flow, Order{ID: "order-100"}); err != nil {
		log.Fatal(err)
	}
}
