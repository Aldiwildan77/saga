package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Aldiwildan77/saga"
)

type Order struct {
	ID string
}

func main() {
	flow := saga.Parallel(
		saga.Do[Order]("payment", func(
			ctx context.Context, o Order,
		) (any, error) {
			fmt.Println("payment started")
			time.Sleep(300 * time.Millisecond)
			fmt.Println("payment completed")
			return nil, nil
		}),

		saga.Do[Order]("shipment", func(
			ctx context.Context, o Order,
		) (any, error) {
			fmt.Println("shipment started")
			time.Sleep(200 * time.Millisecond)
			fmt.Println("shipment completed")
			return nil, nil
		}),

		saga.Do[Order]("notification", func(
			ctx context.Context, o Order,
		) (any, error) {
			fmt.Println("notification started")
			time.Sleep(100 * time.Millisecond)
			fmt.Println("notification completed")
			return nil, nil
		}),
	)

	if err := saga.Run(context.Background(), flow, Order{ID: "order-200"}); err != nil {
		log.Fatal(err)
	}

	fmt.Println("all parallel tasks completed")
}
