# Reflections

This file records what the `tdd-flow` dogfood session taught us about Circuit,
where the current integration helped, where it failed, and what test should come
next.

## What worked

The B-machine layer worked. `tdd-flow.mch` expressed the red-green-refactor
contract cleanly:

- `spec -> red` requires `failingTestObserved = TRUE`.
- `red -> green` requires `testSuitePassed = TRUE`.
- `green -> refactoring` is an explicit choice.
- `refactoring -> green` requires `testSuitePassed = TRUE`.
- `green -> done` requires `testSuitePassed = TRUE`.

The check-binding design also held up. The machine stayed universal and pure;
the project-local proof lived outside B:

- `tdd-flow.mch` defines workflow facts.
- `tdd-flow.checks.yaml` binds facts to check names.
- `check-registry.yaml` chooses this repo's implementations.
- `.bin/circuit-check-tdd-red` proves this repo's red-test evidence.

Session-scoped checks were necessary. Red evidence cannot live at a global path,
because Circuit supports multiple concurrent sessions. The check now requires
`CIRCUIT_SESSION_ID` and reads only:

```text
.tmp/circuit/<session-id>/tdd-red.env
```

No fallback path is allowed. Fallbacks hide bugs and let sessions accidentally
share evidence.

The dogfood also found real runtime issues:

- Externally gated states were incorrectly treated as terminal when all current
  checks were false. That stopped `tdd-flow` at `red`, even though `implement`
  could become enabled after a future passing test suite.
- Blocked diagnostics were too vague (`no disjunct satisfied`). Better
  diagnostics now identify the missing condition, such as
  `failingTestObserved = TRUE`.

## What failed

The agent did not reliably use Circuit just because context was injected.

The session repeatedly showed the same failure mode:

1. Circuit state was visible.
2. The correct transition was available or blocked.
3. The agent still drifted into ordinary coding behavior.
4. The human had to remind the agent to use `circuit_advance`.

That means Pi-hosted Circuit is useful but not sufficient. Context injection and
LLM tools inform the agent, but they do not enforce the workflow. The missing
piece is not another reminder. The missing piece is a driver that makes the
machine the authority for each work turn.

## Architecture insight

The likely next architecture is not only "Pi hosts Circuit" or only "Circuit
drives Pi". The stronger shape is:

```text
Pi -> Circuit -> Pi
```

The outer Pi remains the human interaction surface. A command such as
`/circuit:tdd <task>` starts a Circuit-governed workflow. Circuit then drives an
inner Pi process over RPC for constrained work turns. The inner Pi performs one
state-appropriate slice of work, but Circuit validates whether progress is real.

In that shape:

- Outer Pi is the human UX.
- Circuit is the workflow authority.
- Inner Pi is the replaceable agent runner.

This gives enforcement instead of suggestion.

## UX implication

The raw primitives are too low-level for the desired behavior. The operator
should not have to manually say "use Circuit" at each phase.

A better TDD UX is something like:

```text
/circuit:tdd add status --json
```

or a direct tool equivalent. It should:

1. Start a `tdd-flow` session.
2. Create the session-scoped evidence directory.
3. Drive the agent according to the current state.
4. Require red evidence before `writeTest`.
5. Require green checks before `implement`, `keepGreen`, and `finish`.
6. Re-prompt or reject when the agent attempts state-inappropriate work.

## Test to try next

The next test should exercise the circuit-drives-pi path, still using TDD. The
smallest useful test is not a full production `/circuit:tdd` command yet. It
is a fake-pi integration test proving that Circuit stays in control when the agent
misbehaves.

Proposed red test:

```text
TestRunnerRepromptsAfterBlockedOperation
```

Package:

```text
./internal/circuitrpc
```

Scenario:

1. Start `build-job` in `idle`.
2. Fake Pi first responds with blocked operation `finish`.
3. Circuit must reject it without advancing state.
4. Circuit must send a second prompt for the same `idle` state.
5. Fake Pi then responds with valid operation `start`.
6. Circuit advances `idle -> running`.

Expected assertion:

```text
prompts sent: 2
first state after blocked response: idle
accepted transition: idle -> running
```

Why this test:

- It directly targets process enforcement.
- It uses the existing circuit-drives-pi spike surface.
- It proves the agent cannot make progress by proposing a blocked transition.
- It moves us from passive context injection toward an actual machine-governed
  runner loop.

TDD evidence for the current workflow should be session-scoped:

```text
.tmp/circuit/<tdd-session-id>/tdd-red.env
```

with:

```text
TDD_PACKAGE=./internal/circuitrpc
TDD_RUN=TestRunnerRepromptsAfterBlockedOperation
TDD_EXPECTED_FAIL=TestRunnerRepromptsAfterBlockedOperation
```
