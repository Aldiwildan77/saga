package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Aldiwildan77/saga"
)

type Order struct {
	ID string
}

func main() {
	order := Order{ID: "order-789"}

	flow := saga.Sequence(
		saga.Do[Order, Order]("reserve-inventory", func(
			ctx context.Context, order Order,
		) (Order, error) {
			fmt.Println("reserve inventory")
			return order, nil
		}).Compensate(saga.ParallelCompensations[Order](
			func(context.Context, Order) (any, error) {
				fmt.Println("release inventory")
				return nil, nil
			},
			func(context.Context, Order) (any, error) {
				fmt.Println("notify inventory service")
				return nil, nil
			},
		)),

		saga.Do[Order, Order]("charge-payment", func(
			ctx context.Context, order Order,
		) (Order, error) {
			fmt.Println("charge payment")
			return order, nil
		}).Compensate(saga.SequentialCompensations[Order](
			func(context.Context, Order) (any, error) {
				fmt.Println("record refund")
				return nil, nil
			},
			func(context.Context, Order) (any, error) {
				fmt.Println("refund payment")
				return nil, nil
			},
		)),

		saga.Do[Order, Order]("create-shipment", func(
			ctx context.Context, order Order,
		) (Order, error) {
			fmt.Println("create shipment")
			return order, errors.New("shipping service unavailable")
		}),
	)

	if err := saga.Run(context.Background(), flow, order); err != nil {
		log.Printf("saga failed: %v", err)
	}
}
