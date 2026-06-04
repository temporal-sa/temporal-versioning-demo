---
name: "Pizza workflow waits are activity-side, never workflow.Sleep"
description: "All per-step/drone dwell is simulated as activity execution time; no workflow timers, because Sleep must not appear in the Temporal UI"
type: feedback
---

# Pizza workflow waits are activity-side, never workflow.Sleep

In the pizza workflow, **all dwell/pacing is simulated as activity
execution time — never `workflow.Sleep`**. There must be **no timers in
the workflow history**.

- Per-step work time (`StepDwell` = 15 s) and the drone per-attempt time
  (`droneAttempt` = 5 s) are injectable `Activities` fields (`Dwell`,
  `DroneAttempt`) set by the worker in `Register`; they are **zero in unit
  tests** (tests build `&pizza.Activities{}`) so the suite stays fast.
- Every work step dwells, **including the final `Deliver` step** (the
  "Done" node) — confirmed by the user 2026-06-04.
- The drone v3 retry loop is paced by the `DroneDelivery` activity itself
  (each failing attempt takes `DroneAttempt`); the loop no longer calls
  `workflow.Sleep`.
- The order is marked `state.Done = true` **before** the final `Deliver`
  activity runs, so the completed all-green stepper is visible for that
  activity's dwell **before** the workflow closes — the dashboard lists
  only `ExecutionStatus = 'Running'` workflows, so a finished order drops
  off the board instantly otherwise.

**Why:** The demo is customer-facing; the user explicitly does not want
`Sleep`/`Timer` events cluttering the Temporal Web UI. A `workflow.Sleep`
shows as a timer and reads as a fake workflow step; activity execution
time reads as real processing. The user corrected an earlier attempt that
kept a `workflow.Sleep` for the drone retry backoff.

**How to apply:** Never reintroduce `workflow.Sleep`/`workflow.NewTimer`
in the pizza workflow to pace or delay. To make a step take time, add the
wait inside its activity (context-aware `select` on `ctx.Done()` /
`time.After`, via the `Activities.dwell` helper) and keep the duration an
injectable field that is zero in tests. Guard `make dev` reloads — see
[[make-dev-worker-no-hot-reload]]. Refines the timing / v3 notes in
[[pizza-demo-versioning-approach-and-temporal-wiring]].
