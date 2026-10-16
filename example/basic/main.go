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
	ctx := context.Background()
	order := Order{ID: "order-456"}

	flow := saga.Sequence(
		saga.Do[Order]("validate-order", func(
			ctx context.Context,
			order Order,
		) (any, error) {
			fmt.Println("validate order:", order.ID)
			return nil, nil
		}),

		saga.Nested("prepare-order",
			saga.Do[Order]("reserve-inventory", func(
				ctx context.Context,
				order Order,
			) (any, error) {
				fmt.Println("reserve inventory:", order.ID)
				return nil, nil
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

			saga.Do[Order]("verify-address", func(
				ctx context.Context,
				order Order,
			) (any, error) {
				fmt.Println("verify address:", order.ID)
				return nil, nil
			}),
		),

		saga.Parallel(
			saga.Do[Order]("charge-payment", func(
				ctx context.Context,
				order Order,
			) (any, error) {
				fmt.Println("charge payment:", order.ID)
				return nil, nil
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

			saga.Do[Order]("create-shipment", func(
				ctx context.Context,
				order Order,
			) (any, error) {
				fmt.Println("create shipment:", order.ID)
				return nil, nil
			}),
		),

		saga.Nested("finalize-order",
			saga.Do[Order]("confirm-order", func(
				ctx context.Context,
				order Order,
			) (any, error) {
				fmt.Println("confirm order:", order.ID)
				return nil, nil
			}),
		),
	)

	if err := saga.Run(ctx, flow, order); err != nil {
		log.Fatalf("saga failed: %v", err)
	}

	fmt.Println("order completed:", order.ID)
}
