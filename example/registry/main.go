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
	reg := saga.NewRegistry()

	// Register once, then reuse the same functions across workflows.
	must(saga.Register(reg, "validate", saga.Action[Order, Order](func(_ context.Context, o Order) (Order, error) {
		fmt.Println("validate", o.ID)
		return o, nil
	})))

	must(saga.Register(reg, "charge", saga.Action[Order, Order](func(_ context.Context, o Order) (Order, error) {
		fmt.Println("charge", o.ID)
		return o, nil
	})))

	validate, err := saga.DoFrom[Order, Order](reg, "validate")
	must(err)

	charge, err := saga.DoFrom[Order, Order](reg, "charge")
	must(err)

	flow := saga.Sequence(validate, charge)

	if err := saga.Run(context.Background(), flow, Order{ID: "order-600"}); err != nil {
		log.Fatal(err)
	}

	// Looking up a name that was never registered returns a typed error.
	_, err = saga.DoFrom[Order, Order](reg, "refund")
	if errors.Is(err, saga.ErrorRegistryNotFound) {
		fmt.Println("not found:", err)
	}

	// Looking up with the wrong types returns a type mismatch error.
	_, err = saga.DoFrom[string, string](reg, "validate")
	if errors.Is(err, saga.ErrorRegistryTypeMismatch) {
		fmt.Println("type mismatch:", err)
	}
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
