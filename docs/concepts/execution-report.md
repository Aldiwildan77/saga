# Execution report

Status: planned. Source: automa.

## Goal

Return a structured tree that shows every step, its status, its error, and the
compensation that ran for it. The tree can be logged, stored, or marshalled to
JSON for audits.

## Current behavior

`Run` returns only an error. The internal `Record` tree already holds the
execution data, but it is unexported and has no JSON form.

## API sketch

```go
type Report struct {
    ID           string        `json:"id"`
    Kind         string        `json:"kind"`   // "task", "sequence", "parallel"
    Status       Status        `json:"status"` // "success", "failed", "skipped"
    Error        string        `json:"error,omitempty"`
    Children     []*Report     `json:"children,omitempty"`
    Compensation *Report       `json:"compensation,omitempty"`
    StartedAt    time.Time     `json:"startedAt"`
    Duration     time.Duration `json:"duration"`
}

func RunReport[I any](ctx context.Context, root nodeRunner, input I) (*Report, error)
```

`RunReport` returns the same error as `Run`, plus the tree. The report is
returned even when the run fails, so callers always get the partial picture.

Input and output values are not stored in the report. The report is about
structure and status, not data. Storing data can leak secrets into logs.

## Behavior

- The tree has the same shape as the workflow. Parallel branches appear as
  siblings in the order they were declared, not in the order they finished.
- `Compensation` is set only on steps that were compensated.
- Timestamps come from an injectable clock, so tests are deterministic.
- `Report` implements `json.Marshaler` through plain struct tags. No custom
  marshal code is needed.

## Errors

- `ErrorReportNil`: the caller passes a nil report to a helper that expects one.
  Only used internally.
- Report errors are the same as run errors. The report adds no new failure
  modes.

## Open questions

- Should `Status` be a typed string or an int? Proposal: typed string, so the
  JSON output is readable without a lookup table.
- Automa also marshals to YAML. Proposal: JSON only for now, because the
  library has no YAML dependency.

## Tests and example

- Tests: tree shape for sequence, parallel, and nested groups, compensation
  attached to the right node, JSON output matches the golden file, failed runs
  still return a report.
- Example: `example/report`.
