# Concepts: roadmap plan

This folder holds the design plan for every feature marked **(TBA)** in the
README. Each file covers one feature: the goal, the API sketch, the error
values, the open questions, and where the idea comes from.

All API sketches are generic. Typed values use Go type parameters
(`Node[I, O]`, `Run[I]`, `State[S]`), not `any`, so the compiler checks
the data flow between steps.

## Index

| Concept | Source | Depends on |
| --- | --- | --- |
| [Retry policies](retry-policies.md) | kata | — |
| [Per-step timeout](per-step-timeout.md) | kata | Retry policies (order of attempts) |
| [Parallel fail-fast](parallel-fail-fast.md) | kata | — |
| [Execution hooks](execution-hooks.md) | kata | — |
| [Typed errors](typed-errors.md) | kata | — |
| [Execution modes](execution-modes.md) | automa | — |
| [Execution report](execution-report.md) | automa | Typed errors |
| [Structured logging](structured-logging.md) | automa | Execution hooks |
| [Shared state](shared-state.md) | kata | — |
| [Crash recovery](crash-recovery.md) | automa | Execution report, Idempotent resume |
| [Idempotent resume](idempotent-resume.md) | automa | — |

## Suggested order

1. Typed errors, shared state, execution modes (small, no new dependencies).
2. Retry policies, per-step timeout, parallel fail-fast.
3. Execution hooks, structured logging, execution report.
4. Idempotent resume (documentation and checks), then crash recovery.

## Ground rules for every feature

- No new runtime dependencies. The library uses only the standard library.
- Existing public names and error messages stay the same, unless a concept
  file says otherwise.
- Every feature ships with unit tests, a runnable example under `example/`,
  and a README section. The feature is not "done" until all four exist.
- Compensation keeps running on a context that is not cancelled
  (`context.WithoutCancel`), as it does today.
