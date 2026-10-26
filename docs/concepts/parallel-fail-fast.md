# Parallel fail-fast

Status: planned. Source: kata.

## Goal

When one branch of a `Parallel` group fails, stop the other branches that are
still running. Compensate the branches that already succeeded.

## Current behavior

`Parallel` waits for every branch to finish, then reports all failures joined
into one `ErrorParallelType` error. A slow branch keeps running after another
branch has already failed.

## API sketch

```go
func ParallelFailFast(nodes ...nodeRunner) *Node[any, any]

saga.ParallelFailFast(
    saga.Do[Order, Stock]("reserve-stock", reserveStock),
    saga.Do[Order, Quote]("price-order", priceOrder),
)
```

Generic inputs stay typed. Every branch receives the same `I` value, as it does
now, so `ParallelFailFast` does not change the input type. The return type of
the group is still `[]any`, as it is for `Parallel`.

## Behavior

- The group creates a cancellable child context and gives it to every branch.
- The first branch error cancels that context.
- Branches still running see `ctx.Done()` and should return early.
- Branches that had already succeeded are compensated, as they are today.
- The group returns once every branch has stopped. It never returns while a
  branch is still running.

## Errors

- `ErrorParallelType` still wraps every branch error.
- Branches stopped by the cancel report `context.Canceled`. These are noise,
  so the root error should show the original failure first. Proposal: keep
  them in the joined error, with the first real failure listed first.

## Open questions

- Should `Parallel` itself change to fail-fast? That would break callers who
  rely on every branch finishing. Proposal: keep `Parallel` as it is and add
  `ParallelFailFast`.
- The shared state (see the shared state concept) must be safe under
  cancellation. A branch that is cut off mid-write can leave partial data.

## Tests and example

- Tests: a slow branch stops after a fast branch fails, successful branches
  are compensated, the group does not return early, `errors.Is` finds the
  original error.
- Example: `example/parallel-fail-fast`.
