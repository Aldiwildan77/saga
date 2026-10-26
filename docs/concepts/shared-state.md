# Shared state

Status: planned. Source: kata.

## Goal

Let every step in a workflow read and write one typed state value, next to the
output chaining that already exists. Steps that do not need the state keep
using the chain.

## Current behavior

Each step's output becomes the next step's input (`Sequence`). Values that
several steps need must be passed along the chain or stored with `Yield`, which
keeps one value per type.

## API sketch

```go
type State[S any] struct {
    value *S
    mu    sync.Mutex
}

func NewState[S any](init S) *State[S]
func (s *State[S]) Update(fn func(*S))

func RunWithState[I, S any](ctx context.Context, root nodeRunner, input I, st *State[S]) error

func StateOf[S any](ctx context.Context) (*State[S], bool)
```

A step reads the state with a helper:

```go
saga.Do[Order, Order]("charge-card", func(ctx context.Context, o Order) (Order, error) {
    st, _ := saga.StateOf[Checkout](ctx)
    st.Update(func(c *Checkout) { c.ChargeID = id })
    return o, nil
})
```

`StateOf` is generic, so a step asks for the state type it expects and gets a
compile-time type. A wrong type is a `false` result, not a panic.

## Behavior

- `Update` runs the function under a mutex, so parallel branches can write
  safely. The function must not block for long.
- Compensations receive the same state, so a refund can read the charge ID.
- The state is not copied between steps. Pointers are shared for the whole run.
- Shared state and output chaining work together. A step can read the state
  and still receive the previous step's output as its input.

## Errors

- `ErrorStateMissing`: `StateOf` finds no state for the requested type.
  Returned as the `ok == false` result, plus a sentinel for callers that want
  an error value.

## Relation to yield

`Yield` stores one value per type for the whole run. Shared state stores one
struct that steps update in place. Use `Yield` for a few values passed forward;
use shared state when many steps touch the same record.

## Open questions

- Should `Update` return an error so a step can fail on invalid state changes?
  Proposal: no. Steps return errors themselves.
- Should a `Snapshot` method exist for compensation to read a consistent copy?
  Deferred until a real case needs it.

## Tests and example

- Tests: state is visible across sequential and parallel steps under `-race`,
  compensation reads the state, wrong-type lookup returns false.
- Example: `example/shared-state`.
