# Structured logging

Status: planned. Source: automa.

## Goal

Let the library write its own log lines through the standard `log/slog`
package, with no logging dependency and no output by default.

## API sketch

```go
func WithLogger(l *slog.Logger) RunOption

err := saga.RunWithOptions(ctx, flow, order,
    saga.WithLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil))),
)
```

`RunOption` is a function type used by `RunWithOptions`. `Run` keeps its
signature and uses no logger.

## Behavior

- Default: no logger, so nothing is written. The library uses a discard
  handler internally.
- Levels:
  - `Debug`: step started, step done, with the step name and duration.
  - `Info`: workflow started, workflow completed.
  - `Warn`: a retry happened, or a compensation was started.
  - `Error`: a step failed, or a compensation failed.
- Every line carries the workflow ID and step name as attributes, for example
  `slog.String("step", name)`.
- Logging is built on the execution hooks (see that concept). The logger is one
  hook implementation, so users who want metrics can use hooks without a logger.

## Errors

None. Logging never changes the workflow result. A failing log handler does not
fail the workflow.

## Open questions

- Should attribute names follow a fixed schema (`step`, `duration_ms`,
  `attempt`)? Proposal: yes, and document the schema in the README.
- Go version: the module already targets a recent Go release, so `log/slog` is
  available without a new dependency.

## Tests and example

- Tests: no output by default, expected lines at each level with the right
  attributes, handler errors do not change the result.
- Example: `example/logging`.
