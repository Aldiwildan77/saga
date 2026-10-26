# Per-step timeout

Status: planned. Source: kata.

## Goal

Limit how long one task may run. A task that runs too long fails, and the
normal compensation path runs.

## API sketch

```go
func (n *Node[I, O]) Timeout(d time.Duration) *Node[I, O]

saga.Do[Order, Receipt]("call-gateway", callGateway).
    Timeout(5 * time.Second).
    Retry(3, saga.Exponential(100*time.Millisecond))
```

## Behavior

- The executor passes the task a child context with the deadline set.
  The task must respect `ctx.Done()` to stop early. The executor cannot force
  a goroutine to stop.
- If the task returns after the deadline, its result is discarded and the step
  fails with `context.DeadlineExceeded`.
- Combined with `Retry`, the timeout applies to each attempt, not to the whole
  retry sequence. The retry loop has its own limit (the attempt count).
- A timed-out step is treated as failed. Whether its compensation runs is an
  open question below.

## Errors

- `context.DeadlineExceeded` is wrapped in `StepError` (see the typed errors
  concept), so `errors.Is(err, context.DeadlineExceeded)` works.
- `ErrorInvalidTimeout`: `d <= 0`. Found by `Validate`.

## Open questions

- Compensate a timed-out step or not? A timeout does not prove the side effect
  failed. The safer default is to compensate, and tasks must make their
  compensation idempotent (see the idempotent resume concept). Proposal: compensate.
- Should the timeout also cover compensation? Proposal: no. Compensation uses
  the same rule as today: it runs on a context that is not cancelled.

## Tests and example

- Tests: task that ignores the context is marked failed after the deadline,
  task that honors the context returns `DeadlineExceeded`, timeout per attempt
  with `Retry`.
- Example: `example/timeout`.
