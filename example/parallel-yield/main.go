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

type Stock struct {
	Warehouse string
}

type Quote struct {
	Amount int
}

func main() {
	flow := saga.Sequence(
		// Each branch yields a different type into the shared workflow context.
		// Yield is safe to call from concurrent branches.
		saga.Parallel(
			saga.Do[Order, Stock]("reserve-stock", func(ctx context.Context, order Order) (Stock, error) {
				y := ctx.(saga.Yielder)
				stock := Stock{Warehouse: "wh-east"}
				saga.Yield(y, stock)
				fmt.Println("stock reserved for", order.ID)
				return stock, nil
			}),
			saga.Do[Order, Quote]("price-order", func(ctx context.Context, order Order) (Quote, error) {
				y := ctx.(saga.Yielder)
				quote := Quote{Amount: 4200}
				saga.Yield(y, quote)
				fmt.Println("priced", order.ID)
				return quote, nil
			}),
		),

		// Parallel emits one result per branch, so this step receives []any.
		saga.Do[[]any, bool]("confirm", func(ctx context.Context, _ []any) (bool, error) {
			y := ctx.(saga.Yielder)

			stock, err := saga.YieldValue[Stock](y)
			if err != nil {
				return false, err
			}

			quote, err := saga.YieldValue[Quote](y)
			if err != nil {
				return false, err
			}

			fmt.Printf("confirm from %s at %d\n", stock.Warehouse, quote.Amount)
			return true, nil
		}),
	)

	if err := saga.Run(context.Background(), flow, Order{ID: "order-500"}); err != nil {
		log.Fatal(err)
	}

	fmt.Println("parallel yields collected")
}
