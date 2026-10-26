# Execution hooks

Status: planned. Source: kata.

## Goal

Let callers watch a workflow run (logs, metrics, tracing) without changing
step code.

## API sketch

```go
type Hooks struct {
    OnStepStart          func(ctx context.Context, name string)
    OnStepDone           func(ctx context.Context, name string, d time.Duration)
    OnStepFailed         func(ctx context.Context, name string, err error)
    OnRetry              func(ctx context.Context, name string, attempt int, err error)
    OnCompensationStart  func(ctx context.Context, name string)
    OnCompensationDone   func(ctx context.Context, name string, d time.Duration)
    OnCompensationFailed func(ctx context.Context, name string, err error)
}

func RunWithHooks[I any](ctx context.Context, root nodeRunner, input I, h Hooks) error
```

Hooks are optional. A nil field means "do nothing". The hook names follow kata
so people moving between the libraries see the same names.

## Behavior

- Every hook gets the context of the step it describes, so trace spans can
  attach to the right parent.
- Hooks run synchronously on the step's goroutine. A slow hook slows the
  step, so hooks should be fast or hand work to a channel.
- Parallel branches call hooks concurrently. Hook functions must be safe for
  concurrent use.
- A panic in a hook is not recovered. Hooks are trusted code, like the tasks.

## Errors

Hooks cannot change the workflow result. They return nothing, so there is no
error path to design. The workflow error is returned by `Run` as before.

## Relation to other concepts

- `OnRetry` depends on retry policies. Until retries exist, the hook is not
  called.
- Structured logging (see that concept) is built on these hooks, so the logger
  is one hook implementation.

## Open questions

- Hooks per run (`RunWithHooks`) or per workflow (a builder option)? Proposal:
  per run first, because the root node is immutable and a per-run option is
  simpler to test.

## Tests and example

- Tests: each hook called once per event, names and durations are correct,
  nil hooks are safe, hooks run under `-race` with parallel branches.
- Example: `example/hooks`.
