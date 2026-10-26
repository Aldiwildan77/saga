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
	ctx := context.Background()
	order := Order{ID: "order-456"}

	flow := saga.Sequence(
		saga.Do[Order, Order]("validate-order", func(
			ctx context.Context,
			order Order,
		) (Order, error) {
			fmt.Println("validate order:", order.ID)
			return order, nil
		}),

		saga.Nested("prepare-order",
			saga.Do[Order, Order]("reserve-inventory", func(
				ctx context.Context,
				order Order,
			) (Order, error) {
				fmt.Println("reserve inventory:", order.ID)
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

			saga.Do[Order, Order]("verify-address", func(
				ctx context.Context,
				order Order,
			) (Order, error) {
				fmt.Println("verify address:", order.ID)
				return order, nil
			}),
		),

		saga.Parallel(
			saga.Do[Order, Order]("charge-payment", func(
				ctx context.Context,
				order Order,
			) (Order, error) {
				fmt.Println("charge payment:", order.ID)
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
				ctx context.Context,
				order Order,
			) (Order, error) {
				fmt.Println("create shipment:", order.ID)
				return order, nil
			}),
		),

		saga.Nested("finalize-order",
			// Parallel emits one result per branch, so the next step receives []any.
			saga.Do[[]any, []any]("confirm-order", func(
				ctx context.Context,
				results []any,
			) ([]any, error) {
				fmt.Println("confirm order:", order.ID, "branches:", len(results))
				return results, nil
			}),
		),
	)

	if err := saga.Run(ctx, flow, order); err != nil {
		log.Fatalf("saga failed: %v", err)
	}

	fmt.Println("order completed:", order.ID)
}
