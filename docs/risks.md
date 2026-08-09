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

## 2. Context injection changes agent decisions

The spikes used explicit prompts or manual slash commands. Neither
tested whether injecting circuit state into the system prompt or agent
context actually steers the agent's tool use and reasoning.

Risk: the agent does not meaningfully use circuit state even when it is
in context.

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

## 6. Session persistence across harness restarts

The suspend/resume model works for CLI. The pi extension holds state
via CLI calls. If pi restarts, the suspended file persists but the
extension does not automatically resume the circuit. There is no
session-entry integration.

Risk: active circuits are silently lost on harness restart.
