package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Aldiwildan77/saga/saga"
)

type Order struct {
	ID string
}

func main() {
	order := Order{ID: "order-789"}

	flow := saga.Sequence(
		saga.Do[Order]("reserve-inventory", func(
			ctx context.Context, order Order,
		) error {
			fmt.Println("reserve inventory")
			return nil
		}).Compensate(saga.ParallelCompensations[Order](
			func(context.Context, Order) error {
				fmt.Println("release inventory")
				return nil
			},
			func(context.Context, Order) error {
				fmt.Println("notify inventory service")
				return nil
			},
		)),

		saga.Do[Order]("charge-payment", func(
			ctx context.Context, order Order,
		) error {
			fmt.Println("charge payment")
			return nil
		}).Compensate(saga.SequentialCompensations[Order](
			func(context.Context, Order) error {
				fmt.Println("record refund")
				return nil
			},
			func(context.Context, Order) error {
				fmt.Println("refund payment")
				return nil
			},
		)),

		saga.Do[Order]("create-shipment", func(
			ctx context.Context, order Order,
		) error {
			fmt.Println("create shipment")
			return errors.New("shipping service unavailable")
		}),
	)

	if err := saga.Run(context.Background(), flow, order); err != nil {
		log.Printf("saga failed: %v", err)
	}
}
