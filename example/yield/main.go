package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Aldiwildan77/saga"
)

type Checkout struct {
	Items []string
}

type CheckoutResponse struct {
	ItemIDs []int
}

func main() {
	flow := saga.Sequence(
		saga.Do[Checkout, CheckoutResponse]("checkout", func(ctx context.Context, payload Checkout) (CheckoutResponse, error) {
			y, ok := ctx.(saga.Yielder)
			if !ok {
				return CheckoutResponse{}, fmt.Errorf("context is not yield-enabled")
			}

			response := CheckoutResponse{ItemIDs: []int{1, 2}}
			saga.Yield(y, response)
			fmt.Println("checkout yielded:", response.ItemIDs)
			return response, nil
		}),

		// The previous step's output is the input here; the yielded value is read back by type.
		saga.Do[CheckoutResponse, bool]("send-email", func(ctx context.Context, _ CheckoutResponse) (bool, error) {
			y, ok := ctx.(saga.Yielder)
			if !ok {
				return false, fmt.Errorf("context is not yield-enabled")
			}

			response, err := saga.YieldValue[CheckoutResponse](y)
			if err != nil {
				return false, err
			}

			log.Printf("%v already sent to email", response.ItemIDs)
			return true, nil
		}),
	)

	payload := Checkout{Items: []string{"item-a", "item-b"}}
	if err := saga.Run(context.Background(), flow, payload); err != nil {
		log.Fatal(err)
	}

	fmt.Println("all sequential with yield tasks are completed")
}
