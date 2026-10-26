# Crash recovery

Status: planned. Source: automa.

## Goal

If the process stops in the middle of a workflow, a later run resumes from
where it stopped. Steps that finished are not run again. A rollback that was
in progress is finished.

Crash recovery is opt-in. Without it, nothing is written to disk.

## Current behavior

Everything lives in memory. A process crash loses the `Record` tree, so a
partially finished workflow cannot be resumed or rolled back.

## API sketch

```go
func ResumeRun[I any](ctx context.Context, root nodeRunner, input I, journalPath string) (*Report, error)
```

- If `journalPath` does not exist, the workflow starts fresh and creates the
  journal.
- If it exists, the journal is read and the workflow resumes.

The `Report` type comes from the execution report concept.

## Journal format

An append-only file of JSON lines, one event per line:

```json
{"seq":1,"node":"reserve-stock","event":"started"}
{"seq":2,"node":"reserve-stock","event":"completed","output":{"ReservationID":"r-9"}}
{"seq":3,"node":"charge-card","event":"started"}
```

Compensation events use `"event":"compensating"` and `"event":"compensated"`.

- Each event is written and synced to disk before the next step starts.
- Outputs are stored as JSON, so output types must be JSON-serializable. The
  API documents this and returns `ErrorJournalEncode` for a value that cannot
  be encoded.
- The journal stores the workflow's input, so `ResumeRun` can rebuild the
  steps' inputs.

## Recovery rules

| Last recorded event | Action on resume |
| --- | --- |
| `completed` | Skip the step, reuse the stored output. |
| `started` (no `completed`) | Run the step again. See [Idempotent resume](idempotent-resume.md). |
| `compensating` (no `compensated`) | Run the compensation again. |
| `compensated` | Skip. |
| No event | Run the step. |

## Requirements

- Node names are the journal keys, so every node in a durable workflow needs a
  unique, stable name. `Validate` checks this (see the validation section in
  the README).
- The topology must be the same on resume. If the journal records a node that
  is no longer in the workflow, the run stops with `ErrorJournalTopology`.

## Errors

- `ErrorJournalCorrupt`: a line cannot be parsed, or the sequence numbers have
  a gap.
- `ErrorJournalTopology`: the journal names a node the workflow does not have,
  or the order differs.
- `ErrorJournalEncode`: an output cannot be written as JSON.
- `ErrorJournalIO`: the file cannot be opened, written, or synced. Wraps the
  OS error.

## Open questions

- Journal growth: should the file be compacted after the workflow completes?
  Proposal: delete it on success by default, with an option to keep it.
- Concurrent runs with the same journal path: lock the file? Proposal: yes, a
  lock file, and `ResumeRun` returns `ErrorJournalLocked` if another process
  holds it.
- Parallel branches write events in arbitrary order. The `seq` numbers keep the
  order that was actually written. Recovery must not depend on branch order.

## Tests and example

- Tests: fresh run creates the journal, crash after step N (simulated by
  stopping before the write) resumes at N+1, a `started` step runs again,
  topology mismatch is reported, corrupt line is reported.
- Example: `example/resumable`, which crashes on purpose and then resumes (as
  automa's `resumable_setup` example does).
