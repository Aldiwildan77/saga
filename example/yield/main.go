package yield

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Aldiwildan77/saga"
)

type Checkout struct {
	items []string
}

type CheckoutResponse struct {
	itemIDs []int
}

func main() {

	checkoutStep := saga.Do("checkout", func(ctx context.Context, checkoutPayload Checkout) (CheckoutResponse, error) {
		sCtx := saga.NewYieldContext(ctx)

		response := CheckoutResponse{
			itemIDs: []int{1, 2},
		}

		saga.Yield(sCtx, response)

		return response, nil
	})

	sendEmailStep := saga.Do("send-email", func(ctx context.Context, response CheckoutResponse) (bool, error) {
		if response.itemIDs == nil {
			return false, errors.New("no item ids")
		}

		// send email``
		log.Printf("%v already sent to email", response.itemIDs)
		return true, nil
	})

	flow := saga.Sequence(
		checkoutStep,
		sendEmailStep,
	)

	payload := Checkout{
		items: []string{"item-a", "item-b"},
	}

	if err := saga.Run(context.Background(), flow, payload); err != nil {
		log.Fatal(err)
	}

	fmt.Println("all sequential with yield tasks are completed")
}
