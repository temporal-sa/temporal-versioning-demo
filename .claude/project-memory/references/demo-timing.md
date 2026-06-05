---
name: "Demo timing and the v3 drone regression"
description: "Step/Delivered dwell values and why Delivered is separate; order cadence and ramp steps; v3 drone always-fails with native unlimited retry"
type: project
---

# Demo timing and the v3 drone regression

- Each work step takes ~15 s of **activity** time (`StepDwell`); the final
  `Deliver`/"Done" step has its own `DeliveredDwell` = 7 s. The order generator
  starts one order every ~6 s; the UI ramp increments 10/25/50/100 %.
- **Why `DeliveredDwell` is separate:** the order is marked Done right before
  `Deliver` runs, and the dashboard lists only Running workflows, so the
  all-green card stays on the board during `Deliver`'s dwell. The frontend keeps
  it visible ~4 s (`COLLAPSE_DELAY`) then collapses it; `DeliveredDwell` (7 s)
  is sized to outlast that collapse so the node isn't removed mid-animation. See
  [[frontend-orders-animation]].
- All dwell is activity-side (no workflow timers) — see
  [[workflow-waits-activity-side]].
- **v3 regression:** the Drone delivery activity always fails; the activities use
  Temporal's **native unlimited** retry policy (`RetryPolicy{MaximumAttempts: 0}`,
  with `MaximumInterval` capped to `droneAttempt` so the cadence stays lively), so
  the order stalls **red and stays Running forever — it never ends `Failed`**.
  There is **no** manual retry loop and **no** retry counter; the boolean
  `Failing` flag (set on entry to the drone step) drives the red card. Each failing
  attempt takes ~5 s of activity time (`DroneAttempt`). Activity-retry backoff is
  server-managed, so it still adds no timers to the workflow history — see
  [[workflow-waits-activity-side]].

**Why:** These durations are tuned to the on-screen narrative (orders flow,
ramp is visible, Done cards linger then collapse) and aren't obvious from the
code.

**How to apply:** Treat the dwell values as coupled to the UI collapse timing;
retune both together. See [[worker-versioning-model]].
