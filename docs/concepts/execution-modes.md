# Execution modes

Status: planned. Source: automa.

## Goal

Let a workflow choose what happens after a step fails: roll back the completed
steps (the current behavior), stop without rolling back, or keep running the
remaining steps and collect the errors.

## Naming

The library already has `Mode` for compensation (`SequentialMode`,
`ParallelMode`). The new type is `ExecutionMode`, to keep the two apart.

## API sketch

```go
type ExecutionMode uint8

const (
    RollbackOnError ExecutionMode = iota // default, current behavior
    StopOnError                          // stop, keep completed steps as they are
    ContinueOnError                      // run remaining steps, collect errors
)

func RunWithMode[I any](ctx context.Context, root nodeRunner, input I, m ExecutionMode) error
```

The sketch uses a run option instead of a field on each node. A node that
requires one mode would make it hard to compose nodes into larger workflows.

## Behavior

- `RollbackOnError`: same as `Run` today. On failure, completed steps are
  compensated in reverse order.
- `StopOnError`: on failure, return immediately. No compensation runs. Use it
  when the caller will handle the partial state itself.
- `ContinueOnError`: run every step, even after a failure. Return all failures
  joined. Compensation does not run. Useful for independent checks, such as
  validation of many fields.
- `Run` keeps its current signature and calls `RollbackOnError`.

## Errors

- `ErrorNoneExecutionMode`: unknown mode value. Returned by `RunWithMode`.
- Each mode returns errors in the same shape as today, so `errors.Is` and the
  typed errors concept still apply.

## Open questions

- Is `ContinueOnError` inside a `Sequence` meaningful? A later step may use
  output from a failed step. Proposal: allow it only at the top level first,
  and document the rule for nested groups before adding it there.
- Should a mode be set per group instead of per run? Deferred until a real
  use case asks for it.

## Tests and example

- Tests: each mode on success and failure, compensation runs only in
  `RollbackOnError`, unknown mode error, `Run` behavior unchanged.
- Example: `example/execution-modes`.
