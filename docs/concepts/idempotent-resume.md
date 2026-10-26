# Idempotent resume

Status: planned. Source: automa (durability spec, step-author contract).

## Goal

Make the rules for safe resume explicit. Crash recovery (see that concept) can
run a step twice, so each step and compensation must give the same result when
it runs twice.

## Why this is a concept, not just a doc

Recovery can only guarantee correctness if the user's steps follow the rules.
The library can detect some violations (duplicate node names, a missing
compensation) and can document the rest. This concept covers both.

## The contract

1. **Steps are idempotent.** A step recorded as `started` but not `completed`
   is run again on resume, because the library cannot know whether its side
   effect finished. Running it twice must equal running it once. Check whether
   the resource exists before creating it, and reuse it if it does.
2. **Compensations are idempotent.** A rollback can be retried after a crash
   during the rollback phase.
3. **Resume data lives in state.** Anything a step needs after a restart (IDs,
   handles, earlier outputs) must be in the journaled output or in shared state
   (see that concept). Values held only in closures or struct fields are lost.
4. **Topology is reconstructible.** The code builds the same steps with the
   same names and order on every run. If the steps depend on runtime data,
   store that data so the structure is the same after a restart.

## Library support

- `Validate` checks that node names are unique within a workflow, since the
  journal uses names as keys. Error: `ErrorDuplicateName`.
- Docs: each example that is durable has a comment on which operations are
  idempotent and why.
- The library does not try to detect non-idempotent code. That is not possible
  without running it.

## Example of an idempotent step

```go
saga.Do[Order, Receipt]("charge-card", func(ctx context.Context, o Order) (Receipt, error) {
    // The payment provider returns the existing charge when the same
    // idempotency key is sent again.
    return payments.Charge(ctx, o.ID, o.Amount) // key = order ID
})
```

## Errors

- `ErrorDuplicateName`: two nodes in one workflow share a name. Found by
  `Validate`. This is a new check and must not break workflows that do not use
  crash recovery, so it is a warning-level check unless the workflow is durable.

## Open questions

- Should `Validate` treat duplicate names as an error for all workflows?
  Proposal: yes. Duplicate names already make `Visualize` and error messages
  confusing.
- Can the library offer a helper that stores an idempotency key per step?
  Deferred. The user owns the key format.

## Tests and example

- Tests: `Validate` reports duplicate names, the check ignores non-duplicate
  names, nested groups are checked across levels.
- Example: documented in `example/resumable` (see crash recovery).
