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
- **Yield support**: Share typed values across steps, including parallel branches.
- **Flow visualization**: Print an ASCII tree of workflow structure, including sequential, parallel, nested nodes, and compensation.
- **Cycle detection**: Detect a node that contains itself, so invalid workflow structures are caught before execution.
- **Function registry (TBA)**: Register and resolve reusable task functions by name.
- **Workflow validation (TBA)**: Validate workflow structure before execution, including invalid nodes and malformed configurations.

## Installation

```bash
go get github.com/Aldiwildan77/saga
```

## Quick Start

The following example executes an order workflow with sequential validation, a nested reservation, parallel payment and shipment operations, and a final confirmation step.

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
    order := Order{ID: "order-456"}

    flow := saga.Sequence(
        saga.Do[Order, Order]("validate-order", func(
            ctx context.Context, order Order,
        ) (Order, error) {
            fmt.Println("validate order:", order.ID)
            return order, nil
        }),

        saga.Parallel(
            saga.Do[Order, Order]("charge-payment", func(
                ctx context.Context, order Order,
            ) (Order, error) {
                fmt.Println("charge payment:", order.ID)
                return order, nil
            }),

            saga.Do[Order, Order]("create-shipment", func(
                ctx context.Context, order Order,
            ) (Order, error) {
                fmt.Println("create shipment:", order.ID)
                return order, nil
            }),
        ),

        // Parallel emits one result per branch, so the next step receives []any.
        saga.Do[[]any, []any]("confirm-order", func(
            ctx context.Context, results []any,
        ) ([]any, error) {
            fmt.Println("confirm order:", order.ID, "branches:", len(results))
            return results, nil
        }),
    )

    if err := saga.Run(context.Background(), flow, order); err != nil {
        log.Fatalf("saga failed: %v", err)
    }

    fmt.Println("order completed:", order.ID)
}
```

## Core Concepts

### Tasks

`Do[I, O]` defines an individual operation. Each task receives a context and its typed input, and returns its typed output along with an error if the operation fails.

```go
saga.Do[Order, Order]("validate-order", func(
    ctx context.Context, order Order,
) (Order, error) {
    return order, nil
})
```

### Data Flow Between Steps

Each step's output becomes the next step's input, so the types must line up:

- In a `Sequence`, the output of one child is the input of the next child.
- A `Nested` group behaves like a sequence: its output is the output of its last child.
- A `Parallel` group gives every branch the same input and returns a `[]any` of the branch outputs. The step after a parallel group must accept `[]any`.

If the input type does not match, the run fails with an input type mismatch error (see [Error Handling](#error-handling)).

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

`Parallel` executes its child nodes concurrently and waits for all branches to finish. If a branch fails, the parallel group returns an error that includes every branch failure.

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

`Run` wraps the workflow context with a yield-capable context so tasks can store typed values for later retrieval. Use `Yield` to attach a value to the current workflow and `YieldValue` to read it back in a downstream step. Values are keyed by type, so each type holds one value; yielding the same type again overwrites the previous one.

```go
flow := saga.Sequence(
    saga.Do[Order, Order]("lookup-customer", func(ctx context.Context, order Order) (Order, error) {
        y, ok := ctx.(saga.Yielder)
        if !ok {
            return Order{}, fmt.Errorf("context is not yield-enabled")
        }

        customer := Customer{ID: "cust-42"}
        saga.Yield(y, customer)
        return order, nil
    }),
    saga.Do[Order, string]("ship-order", func(ctx context.Context, order Order) (string, error) {
        y, ok := ctx.(saga.Yielder)
        if !ok {
            return "", fmt.Errorf("context is not yield-enabled")
        }

        customer, err := saga.YieldValue[Customer](y)
        if err != nil {
            return "", err
        }

        fmt.Println("ship to:", customer.ID)
        return customer.ID, nil
    }),
)
```

Yielded values are visible to later steps in the same workflow, including steps after a nested group. They are also safe to yield from parallel branches. A value yielded inside a parallel branch is available to later steps, but which branch's value wins when two branches yield the same type is not defined, so give each concurrent branch its own type.

A nested `Run` that receives a yield-capable context can read values yielded by its parent. Values yielded by the nested run stay in the nested scope and do not leak back to the parent.

### Compensation

`Compensate` attaches compensating actions to a task. When a workflow fails, the executor uses the execution journal to roll back successful work. Failed tasks are not compensated.

```go
saga.Do[Order, Order]("charge-payment", chargePayment).
    Compensate(saga.SequentialCompensations[Order](
        recordRefund,
        refundPayment,
    ))
```

Sequential compensations run in reverse registration order. In this example, `refundPayment` runs before `recordRefund`. All compensations run even if one fails, and their errors are joined.

For independent compensations that can safely run concurrently, use `ParallelCompensations`:

```go
saga.Do[Order, Order]("reserve-inventory", reserveInventory).
    Compensate(saga.ParallelCompensations[Order](
        releaseInventory,
        notifyInventoryService,
    ))
```

Rollback runs on a context that is not cancelled, so compensations still run after the workflow context has been cancelled.

Compensation is a business operation, not a database rollback. Each compensating action should handle its own failures and should ideally be safe to retry.

### Flow Visualization

`Visualize` returns an ASCII tree of a workflow without running any step. Sequences, parallel groups, nested groups, and compensation plans are all shown.

```go
tree, err := saga.Visualize(flow)
if err != nil {
    log.Fatal(err)
}

fmt.Print(tree)
```

```text
sequence
├── validate-order
├── nested prepare-order
│   ├── reserve-inventory [compensate: parallel x2]
│   └── verify-address
└── parallel
    ├── charge-payment [compensate: sequential x1]
    └── create-shipment
```

`Visualize` returns `ErrorRootEmpty` for a nil root, `ErrorNodeNoExecuteFunc` for a task without an execute function, and `ErrorNoneKind` or `ErrorNoneMode` for unsupported node kinds or compensation modes.

### Cycle Detection

`DetectCycles` reports `ErrorCycle` when a node appears inside its own subtree. It walks the workflow without running any step.

```go
if err := saga.DetectCycles(flow); err != nil {
    log.Fatal(err)
}
```

Reusing one node in several places is not a cycle. Only a node that contains itself is reported. Because the builder API composes nodes bottom-up, a cycle can't be built with the public API today; this check protects against invalid structures and makes the workflow walk safe to extend.

### Error Handling

`Run` returns an error when the workflow fails. The returned error can contain both the execution failure and any compensation failures.

Use `errors.Is` to check for known errors:

```go
if errors.Is(err, saga.ErrorSequenceType) {
    // A sequential workflow failed.
}
```

| Error | When it is returned |
| --- | --- |
| `ErrorRootEmpty` | `Run` is called with a nil root node. |
| `ErrorNodeNoExecuteFunc` | A task was created with a nil execute function. |
| `ErrorNodeExecutedFunc` | A task's function returned an error. Wraps the original error with the step name. |
| `ErrorSequenceType` | A child of a `Sequence` or `Nested` group failed, or its input type did not match. |
| `ErrorParallelType` | One or more branches of a `Parallel` group failed. Every branch error is included. |
| `ErrorNoneKind` | A node has an unsupported kind. |
| `ErrorCompensation` | A compensating action failed. Wraps the original error with the step name. |
| `ErrorNoneMode` | A compensation plan has an unsupported mode. |
| `context.Canceled` / `context.DeadlineExceeded` | The context was cancelled or timed out before a step ran. |

Input type mismatches are reported as `input type mismatch` in the error message and are wrapped by `ErrorSequenceType` when they happen inside a sequence.

Tasks should return meaningful errors so callers can identify the underlying cause.

## Examples

The repository includes runnable examples:

| Example | Description |
| --- | --- |
| `example/basic` | Complete workflow happy path |
| `example/compensation` | Failure and compensating actions |
| `example/errors` | Every error returned by `Run`, and how to detect it with `errors.Is` |
| `example/nested` | Nested workflow composition |
| `example/nested-yield` | Nested workflows with yield propagation |
| `example/parallel` | Concurrent task execution |
| `example/flow` | Print a workflow's structure with `Visualize` |
| `example/cycle` | Check a workflow for cycles with `DetectCycles` |
| `example/parallel-yield` | Yielding typed values from concurrent branches |
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
