# Retry policies

Status: planned. Source: kata.

## Goal

Retry a failed task a fixed number of times, with a delay between attempts.
Retries happen inside the task. Compensation runs only after the last attempt
fails.

## API sketch

```go
type Backoff interface {
    Delay(attempt int) time.Duration
}

func Fixed(d time.Duration) Backoff
func Exponential(base time.Duration) Backoff // ceiling of 5 minutes
func Jitter(b Backoff) Backoff
func Cap(b Backoff, max time.Duration) Backoff
var NoDelay Backoff

func (n *Node[I, O]) Retry(attempts int, policy Backoff) *Node[I, O]

saga.Do[Order, Receipt]("charge-card", chargeCard).
    Retry(3, saga.Jitter(saga.Exponential(100*time.Millisecond))).
    Compensate(saga.SequentialCompensations[Order](refundCard))
```

`Retry` returns a copy, like `Compensate` does today. `attempts` counts all
tries, including the first. Each retry calls the same generic function with
the same typed input `I`.

## Behavior

- Attempt 1 runs immediately. Before attempt `k` (k ≥ 2) the executor waits
  `policy.Delay(k-1)`.
- Waiting respects the context. If the context is cancelled during the wait,
  the step stops and returns the context error.
- If every attempt fails, the step fails with the last error. Compensation
  then runs once, as it does now.
- `Exponential` caps the delay at 5 minutes so large attempt counts do not
  overflow.

## Errors

- `ErrorRetryExhausted`: all attempts failed. Wraps the last cause. Works with
  `errors.Is` for both `ErrorRetryExhausted` and the original error.
- `ErrorInvalidRetry`: `attempts < 1` or a nil policy. Found by `Validate`
  (see the validation concept in the README).

## Open questions

- Retry only on some errors (for example, not on validation errors). kata does
  not have this. Proposal: `RetryIf(func(error) bool)` as a later option.
- Should `Jitter` use a seeded random source so tests are deterministic?
  Proposal: yes, accept an `io.Reader` or `*rand.Rand` option.

## Tests and example

- Tests: attempt count, delay sequence, cancellation during the wait,
  exhaustion error, no compensation until the final failure.
- Example: `example/retry`.
