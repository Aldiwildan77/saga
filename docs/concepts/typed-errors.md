# Typed errors

Status: planned. Source: kata.

## Goal

Let callers read which step failed, and whether a compensation also failed,
without parsing error strings. Keep `errors.Is` working.

## Current behavior

Errors are built with `fmt.Errorf` and sentinel values. A caller can match
`ErrorSequenceType` or `ErrorCompensation`, but cannot get the step name
without parsing the message.

## API sketch

```go
type StepError struct {
    StepName string
    Cause    error
}

func (e *StepError) Error() string
func (e *StepError) Unwrap() error

type CompensationFailure struct {
    StepName string
    Err      error
}

type CompensationError struct {
    Failed []CompensationFailure
}

func (e *CompensationError) Error() string
func (e *CompensationError) Unwrap() []error
```

Usage:

```go
var stepErr *saga.StepError
if errors.As(err, &stepErr) {
    log.Printf("step %q failed: %v", stepErr.StepName, stepErr.Cause)
}

var compErr *saga.CompensationError
if errors.As(err, &compErr) {
    for _, f := range compErr.Failed {
        alert(f.StepName, f.Err)
    }
}
```

`Unwrap() []error` lets `errors.Is` reach every cause, which matches the
`errors.Join` behavior the library already uses.

## Behavior

- A failed task produces a `*StepError`. The message stays the same as today:
  `step "charge" failed: card declined`.
- Compensation failures are collected into one `*CompensationError`, one entry
  per failed compensation.
- The existing sentinel values stay. `errors.Is(err, ErrorSequenceType)` still
  works, because the wrapping chain is unchanged.

## Errors

No new sentinel values. The types are the new part.

## Open questions

- Should `StepError` expose the node kind (task, sequence, parallel)? Proposal:
  no for now. The execution report (see that concept) carries that detail.
- Changing the return type of internal helpers is an internal refactor, not a
  public API change. Confirm that with the existing tests before merging.

## Tests and example

- Tests: `errors.As` finds the step name at each nesting level, compensation
  failures are all listed, messages are unchanged, `errors.Is` still matches
  the sentinels.
- Example: `example/typed-errors`.
