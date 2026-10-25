# Saga

A lightweight, generic Saga workflow library for Go. Build workflows from composable tasks, execute them sequentially or in parallel, and define compensating actions to handle failures.

## Features

- **Generic workflows**: Pass typed input through every task.
- **Sequential execution**: Execute tasks in a defined order.
- **Parallel execution**: Run independent tasks concurrently.
- **Nested workflows**: Compose smaller workflows into larger ones.
- **Compensation**: Execute compensating actions when a workflow fails.
- **Execution journal**: Track execution results and successful steps.
- **Context support**: Respect cancellation during execution.
- **Error propagation**: Preserve underlying errors for inspection with `errors.Is`.
- **Yield support**: Support yielding or pausing workflow execution for future continuation.
- **Flow visualization (TBA)**: Generate an ASCII representation of workflow structure, including sequential, parallel, and nested nodes.
- **Cycle detection (TBA)**: Detect cyclic workflow references to prevent invalid workflow structures.
- **Function registry (TBA)**: Register and resolve reusable task functions by name.
- **Workflow validation (TBA)**: Validate workflow structure before execution, including invalid nodes and malformed configurations.

## Installation

```bash
go get github.com/Aldiwildan77/saga
```

## Quick Start

The following example executes an order workflow with sequential validation, parallel payment and shipment operations, and a final confirmation step.

```go
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
    flow := saga.Sequence(
        saga.Do[Order]("validate-order", func(
            ctx context.Context, order Order,
        ) error {
            fmt.Println("validate order:", order.ID)
            return nil
        }),

        saga.Parallel(
            saga.Do[Order]("charge-payment", func(
                ctx context.Context, order Order,
            ) error {
                fmt.Println("charge payment:", order.ID)
                return nil
            }),

            saga.Do[Order]("create-shipment", func(
                ctx context.Context, order Order,
            ) error {
                fmt.Println("create shipment:", order.ID)
                return nil
            }),
        ),

        saga.Do[Order]("confirm-order", func(
            ctx context.Context, order Order,
        ) error {
            fmt.Println("confirm order:", order.ID)
            return nil
        }),
    )

    order := Order{ID: "order-456"}

    if err := saga.Run(context.Background(), flow, order); err != nil {
        log.Fatalf("saga failed: %v", err)
    }

    fmt.Println("order completed:", order.ID)
}
```

## Core Concepts

### Tasks

`Do` defines an individual operation. Each task receives a context and the workflow's typed input, and returns an error if the operation fails.

```go
saga.Do[Order]("validate-order", func(
    ctx context.Context, order Order,
) error {
    return nil
})
```

### Sequential Workflows

`Sequence` executes its child nodes in order. Execution stops when a child fails.

```go
saga.Sequence(
    validateOrder,
    reserveInventory,
    chargePayment,
)
```

### Parallel Workflows

`Parallel` executes its child nodes concurrently and waits for all branches to finish. If a branch fails, the parallel group returns an error.

```go
saga.Parallel(
    chargePayment,
    createShipment,
    sendNotification,
)
```

Use parallel execution only when the operations can safely run independently.

### Nested Workflows

`Nested` groups tasks into a named sequential workflow, making larger workflows easier to organize.

```go
saga.Nested("prepare-order",
    reserveInventory,
    verifyAddress,
)
```

### Yielded Values

`Run` wraps the workflow context with a yield-capable context so tasks can store typed values for later retrieval. Use `Yield` to attach a value to the current workflow and `YieldValue` to read it back in a downstream step.

```go
flow := saga.Sequence(
    saga.Do[Order]("lookup-customer", func(ctx context.Context, order Order) (Customer, error) {
        y, ok := ctx.(saga.Yielder)
        if !ok {
            return Customer{}, fmt.Errorf("context is not yield-enabled")
        }

        customer := Customer{ID: "cust-42"}
        saga.Yield(y, customer)
        return customer, nil
    }),
    saga.Do[Order]("ship-order", func(ctx context.Context, order Order) (bool, error) {
        y, ok := ctx.(saga.Yielder)
        if !ok {
            return false, fmt.Errorf("context is not yield-enabled")
        }

        customer, err := saga.YieldValue[Customer](y)
        if err != nil {
            return false, err
        }

        fmt.Println("ship to:", customer.ID)
        return true, nil
    }),
)
```

This pattern works for nested workflows too: a task in a nested group can yield a value that a later sibling or parent step reads from the same workflow context.

### Compensation

`Compensate` attaches compensating actions to a task. When a workflow fails, the executor uses the execution journal to roll back successful work.

```go
saga.Do[Order]("charge-payment", chargePayment).
    Compensate(saga.SequentialCompensations[Order](
        recordRefund,
        refundPayment,
    ))
```

Sequential compensations run in reverse registration order. In this example, `refundPayment` runs before `recordRefund`.

For independent compensations that can safely run concurrently, use `ParallelCompensations`:

```go
saga.Do[Order]("reserve-inventory", reserveInventory).
    Compensate(saga.ParallelCompensations[Order](
        releaseInventory,
        notifyInventoryService,
    ))
```

Compensation is a business operation, not a database rollback. Each compensating action should handle its own failures and should ideally be safe to retry.

### Error Handling

`Run` returns an error when the workflow fails. The returned error can contain both the execution failure and any compensation failures.

Use `errors.Is` to check for known errors:

```go
if errors.Is(err, saga.ErrorSequenceType) {
    // A sequential workflow failed.
}
```

Tasks should return meaningful errors so callers can identify the underlying cause.

## Examples

The repository includes runnable examples:

| Example | Description |
| --- | --- |
| `example/basic` | Complete workflow happy path |
| `example/compensation` | Failure and compensating actions |
| `example/nested` | Nested workflow composition |
| `example/nested-yield` | Nested workflows with yield propagation |
| `example/parallel` | Concurrent task execution |
| `example/yield` | Typed value sharing across workflow steps |

Run an example:

```bash
go run ./example/basic
```

Run the test suite:

```bash
go test ./...
```

Run tests with the race detector:

```bash
go test -race ./...
```

## Design Principles

- Keep workflows composable and explicit.
- Use typed input rather than unstructured payloads.
- Separate execution logic from compensation logic.
- Execute independent work concurrently only when appropriate.
- Preserve execution errors and compensation errors for diagnosis.
