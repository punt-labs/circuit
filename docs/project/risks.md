# Risks

This document separates evidence already gathered from risks that remain open.
It is not a milestone checklist. Accepted design decisions live in
[`../architecture/decisions.md`](../architecture/decisions.md); operational
contracts live in [`../operations/run-contract.md`](../operations/run-contract.md).

## 1. Guided outcomes may not be better

**Evidence:** Circuit has guided a real agent through the multi-step `tdd-flow`,
including external test and code-quality checks, blocked progress, and
refactoring loops. Driver traces make the sequence observable. Pi-hosted use has
also shown an agent reading injected state and requesting valid transitions.

**Remaining risk:** The evidence establishes feasibility, not comparative
benefit. Circuit still needs controlled comparisons of guided and unguided runs
on realistic work: outcome quality, supervision burden, recovery cost, and time
to completion.

## 2. Context injection is guidance, not enforcement

**Evidence:** The pi extension injects active session state and valid operations
through `before_agent_start`; agents have used that context to call Circuit tools
without an explicit reminder.

**Remaining risk:** An agent can ignore injected guidance, especially under a
competing goal or when several operations are enabled. The enforced path is
`circuit drive`, where Circuit owns state and accepts or rejects requested
events. Pi-hosted context and tools should not be described as equivalent
enforcement.

## 3. External observations may be too coarse

**Evidence:** Machines bind B BOOL variables to registered commands. Loading
rejects incomplete bindings, scaffolding creates safe failing stubs, and check
results and invocation history persist with the session. Local quality commands
have been exercised in real guided runs.

**Remaining risk:** Production workflows need heterogeneous and potentially
slow evidence such as CI state, review threads, and artifact analysis. The
current command-check model may need stronger provenance, timeout behavior, and
freshness semantics before those facts can safely authorize transitions.

## 4. Branch choice and event extraction need broader evidence

**Evidence:** `tdd-flow` branches between finishing and refactoring according to
an external quality fact, and Circuit rejects invalid or premature events. The
driver can re-prompt when no event is extracted or an event is blocked.

**Remaining risk:** Evidence is narrow. More workflows should test several
simultaneously valid choices, ambiguous agent responses, and decisions requiring
new observations rather than a single boolean command result.

## 5. Recovery is bounded but not fully operable

**Evidence:** `retry-flow` proves that failed checks can block a transition,
persist their invocation state, and later allow a retry. Driven workflows can
re-prompt the same state, and operators can inspect status, stop sessions, and
unload stopped sessions.

**Remaining risk:** Repeated failure can still exhaust retry bounds or leave a
non-terminal workflow with no currently enabled operation. Circuit lacks a
first-class intervention protocol for amending evidence, changing guidance, or
escalating a stuck session while preserving an auditable history.

## 6. Concurrent session storage is not transactional

**Evidence:** Sessions have explicit lifecycle states, persist under
`.tmp/sessions/`, survive short-lived CLI invocations, and can run independently
by session ID. Terminal completion stops only the completed session.

**Remaining risk:** Storage is repository-local and lacks locking for truly
concurrent writers. Harness-native session association, human-friendly naming,
selection UX, and robust recovery from interrupted writes remain open.

## 7. Workspace readiness can invalidate check meaning

**Evidence:** Driver traces exposed a run where a missing development dependency
caused the main check to fail for an environment reason, which superficially
looked like valid red-phase evidence.

**Remaining risk:** A command's exit status does not by itself prove that it
failed for the intended reason. Guided workflows need explicit workspace
preflight and, where authorization depends on failure semantics, stronger check
provenance than success/failure alone.
