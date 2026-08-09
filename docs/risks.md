# Risks

Unproved risks after spikes 1–3. Each needs either a spike, a test, or
a design decision before circuit can claim usefulness.

## 1. Agent behavior actually improves

No spike tested whether an agent constrained by a circuit machine
produces better workflow outcomes than an unconstrained agent. The
spike 3 agent was trivially compliant: it received "say the event
name" and said it. A real agent doing real work has not been tested
against a circuit machine.

Risk: the machine is correct but the agent ignores or works around it.

**Milestone 4 update:** the agent called `circuit_status` and
`circuit_advance` tools unprompted when asked to advance. It did not
attempt to bypass the machine. This is a positive signal but only for
a trivial machine (`build-job`). A real multi-step workflow with
ambiguous choices has not been tested.

## 2. Context injection changes agent decisions

The spikes used explicit prompts or manual slash commands. Neither
tested whether injecting circuit state into the system prompt or agent
context actually steers the agent's tool use and reasoning.

Risk: the agent does not meaningfully use circuit state even when it is
in context.

**Milestone 4 update:** context injection is now implemented via
`before_agent_start`. The agent received circuit state in context and
used the `circuit_advance` tool to progress `build-job` from `idle` to
`running` on the first attempt. The injection included enabled/blocked
operations and a guideline to use `circuit_advance`. The agent
complied. Not yet tested whether injection changes behavior when the
agent has a competing goal or when multiple operations are enabled.

## 3. Facts from real observations

`review-flow` binds `makeCheckPassed` to `make check`, which works.
Real workflows need richer facts: PR check status, review thread
counts, file change analysis. The check-binding model has not been
tested with multiple heterogeneous observations or with latency
concerns.

Risk: check bindings are too slow or too coarse for real workflows.

## 4. Multi-step workflows with branches

No spike tested a machine with multiple enabled transitions where the
agent must choose correctly based on gathered evidence. That is the
scenario where circuit should provide the most value.

Risk: operation extraction does not work when choices are ambiguous or
require reasoning.

## 5. Error recovery and stuck states

No spike tested what happens when the agent makes a mistake, when a
check fails repeatedly, or when the machine reaches a non-terminal
state with no enabled transitions under the current facts. The runtime
reports "blocked" and stops.

Risk: real workflows stall and the human has no circuit-level recovery
path.

**Milestone 4 update:** `retry-flow` machine with an alternating
check proves the block/retry loop works. First `advance proceed` is
blocked with `gateOpen: FALSE (invocations: 1)`. Second `advance
proceed` passes with `gateOpen: TRUE (invocations: 2)` and advances
`waiting -> done`. The runtime preserves check state and invocation
counts across suspend/resume boundaries. Tested at Go runtime, CLI,
and ProB levels. Not yet tested with an agent driving the retry loop
automatically.

## 6. Session persistence across harness restarts

The suspend/resume model works for CLI. The pi extension holds state
via CLI calls. If pi restarts, the suspended file persists but the
extension does not automatically resume the circuit. There is no
session-entry integration.

Risk: active circuits are silently lost on harness restart.
