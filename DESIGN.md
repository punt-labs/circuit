# Design

Architectural decisions for `circuit`. Each ADR records the decision,
the evidence that supports it, and the constraints it creates.

Formal design validation now starts in `docs/spec/circuit-runtime.tex`, a Z
specification of Circuit's totalized load/scaffold/start/advance/stop/context
operations.

## ADR 1: B-Method abstract machines as the formal model

**Decision:** Circuit workflow definitions are B abstract machines
(`.mch` files). The B machine is the authored artifact, the reviewed
artifact, and the runtime artifact.

**Evidence:** Spike 1 proved Go can parse and evaluate `.mch` files at
runtime without ProB. ProB validates the same files during development.
Four machines exist and pass both ProB model-check and Go runtime
evaluation.

**Constraints:**

- ProB is a development/release dependency, not a runtime dependency.
- The `.mch` file is the source of truth. No generated JSON or YAML
  intermediate is the primary format.
- Machine authors must stay within the Circuit-B profile.

## ADR 2: Circuit-B profile — a strict B subset

**Decision:** The Go runtime interprets a deliberately small subset of
B, not all B. Valid B outside the profile is rejected with actionable
diagnostics.

**Evidence:** The parser supports enumerated sets, NAT, BOOL, PRE/THEN,
IF/ELSIF/ELSE, assignment, parallel assignment, and membership
predicates. Unsupported constructs like ANY are detected and rejected
with profile-specific error messages. ProB accepts the full B language;
Circuit accepts only the profile.

**Constraints:**

- Every profile expansion requires a design decision, not just a parser
  change.
- The profile must remain small enough that the Go evaluator can be
  trusted without a formal proof of its own.

## ADR 3: Multi-pass parser architecture

**Decision:** The Circuit-B parser uses four explicit passes: lex/parse,
resolve names/types, validate profile, evaluate. Passes are not merged.

**Evidence:** The txt2tex experience showed that single-pass parsers
accumulate context flags and hidden coupling. The multi-pass design
keeps each concern isolated. Parser technology (hand-written,
participle, pigeon) can replace pass 1 without affecting passes 2–4.

**Constraints:**

- Every token and AST node carries a source span.
- Name resolution does not happen during parsing.
- Profile validation does not happen during resolution.
- Parser libraries may replace pass 1 only.

## ADR 4: Check bindings — B booleans bound to external commands

**Decision:** Runtime preconditions that depend on the outside world are
represented as B boolean variables. A companion `.checks.yaml` file
binds each variable to a registered check command. B never contains
shell commands. A machine with BOOL variables must load with complete
bindings before it can start.

**Evidence:** `review-flow.mch` requires `makeCheckPassed = TRUE`.
`review-flow.checks.yaml` binds it to `makeCheck` in
`check-registry.yaml`. `retry-flow.mch` uses an alternating check to
prove the block/retry loop. `scaffold` can generate missing bindings
and false registry stubs from resolved BOOL variables. These paths are
covered by Go runtime and CLI tests.

**Constraints:**

- B remains pure and ProB-checkable.
- Commands live in the check registry, not in B.
- Every BOOL variable must have a binding before the machine can load.
- Every bound check must reference a registry entry with a compatible
  return type.
- `scaffold` may generate missing bindings and registry stubs, but the
  stubs default to `false` so incomplete integrations block safely.
- Checks run before advance and their results (boolean + invocation
  count) are persisted in the session.

## ADR 5: Session lifecycle

**Decision:** Each circuit run is a session with explicit lifecycle
states:

```text
SessionState ::= unloaded | active | suspended | stopped
```

A runtime may hold multiple active sessions. When advance reaches a
terminal state, only that session auto-stops.

**Evidence:** Without auto-stop, terminal machines injected stale state
into agent context indefinitely. The session lifecycle fix was driven by
a real usability problem observed during milestone 4 testing.

**Constraints:**

- `status` reports known active or stopped sessions.
- `advance` requires at least one active session.
- `status` with no known session is informational, not an error.
- Implicit `advance` and `stop` are valid only when exactly one session
  is active; otherwise the caller must provide a session ID.
- Stopped sessions remain known so `stop` is idempotent for known sessions.
- Unloaded/unknown sessions cannot be stopped.
- Context injection fires only when at least one session is active.

## ADR 6: Multiple concurrent sessions

**Decision:** The runtime supports multiple active sessions. Each
session is independent: its own machine, its own B state, and its own
check history. Session IDs use `<machine>-<4hex>` such as
`build-job-a3f8`.

**Evidence:** Real workflows require concurrent machines. A PR workflow
may run `review-flow` and `pr-watch` simultaneously — one tracking
code review lifecycle, the other tracking CI status. Runtime and CLI
tests now cover starting two sessions, listing both, targeted advance,
implicit ambiguity errors, targeted stop, and per-session terminal
auto-stop.

**Constraints:**

- Sessions are identified by generated session ID, not by machine name
  alone.
- Active sessions are persisted under `.tmp/sessions/<id>.json`.
- Context injection includes all active sessions.
- CLI commands accept optional session qualifiers where ambiguity is
  possible.

## ADR 7: Suspend/resume as implicit CLI lifecycle

**Decision:** Short-lived CLI commands implicitly resume from a
suspended file, operate in memory, and suspend on exit. The suspended
file is a serialization artifact, not the conceptual source of state.

**Evidence:** Every CLI invocation since spike 1 has used this pattern.
The `retry-flow` block/retry test proves check state and invocation
counts survive suspend/resume boundaries correctly.

**Constraints:**

- A future long-running process should keep the same runtime in memory
  and suspend only on exit.
- The suspended file format is JSON, stored under `.tmp/sessions/`.
- `.tmp/circuit.suspended.json` is read only as a legacy migration
  source.

## ADR 8: Pi hosts circuit as a thin adapter

**Decision:** The pi extension shells out to the Go CLI for every
machine decision. No transition logic lives in TypeScript.

**Evidence:** Spike 2 proved this works. The extension registers slash
commands and LLM tools that call the CLI and parse its output. The
extension does not know about states, transitions, or preconditions.

**Constraints:**

- The extension must not duplicate or interpret machine semantics.
- All machine intelligence stays in Go.
- TypeScript owns only: command parsing, context formatting, output
  parsing, and pi API integration.

## ADR 9: Circuit can drive pi over RPC

**Decision:** Circuit can own machine state and drive `pi --mode rpc`
as an agent backend. The Go runner sends prompts, observes
`agent_settled`, extracts the agent's chosen operation, and validates
it against the B machine.

**Evidence:** Spike 3 completed a full `idle -> running -> done` loop
on `build-job` with circuit as the outer runner and pi as the agent
backend.

**Constraints:**

- The RPC protocol logic is in `internal/circuitrpc`, tested with a
  fake-pi integration test.
- The spike runner is at `cmd/circuit-rpc-spike/`, not production code.
- This relationship is viable for headless/supervised automation but is
  not the primary interactive model.

## ADR 10: LLM tools with full slash command parity

**Decision:** Every slash command has a corresponding LLM tool. The
agent can discover, load, scaffold, start, operate, and stop circuits
without human intervention.

**Evidence:** Milestone 4 proved this. The agent called `circuit_status`
then `circuit_advance` unprompted and progressed `build-job` from
`idle` to `running`. All seven tools are registered; parsing,
formatting, and Go command behavior are tested.

**Constraints:**

- Tool names use underscores: `circuit_list`, `circuit_load`,
  `circuit_scaffold`, `circuit_start`, `circuit_status`,
  `circuit_advance`, `circuit_stop`.
- Slash commands use the `/circuit <verb>` namespace.
- Both call the same Go CLI underneath.

## ADR 11: Context injection via before_agent_start

**Decision:** When at least one session is active, the pi extension
injects every active session's current machine state and valid
operations into the agent's context on every turn via the
`before_agent_start` hook.

**Evidence:** Milestone 4 proved the agent used the injected context to
choose the correct tool call without being explicitly told to check
circuit state.

**Constraints:**

- No injection when no session is active.
- No injection for stopped sessions (terminal auto-stop).
- The injection is a custom message with `display: false` so it
  participates in LLM context but does not clutter the TUI.

## ADR 12: Automated testing pyramid with coverage targets

**Decision:** Every tier of the testing pyramid is automated. No
manual-only tests. Core packages maintain ≥85% statement coverage.

**Evidence:** All tiers are implemented and pass in `make check`:

- Go unit tests: circuitb 85%, circuitrun 85%, circuitrpc 97%
- Go CLI tests: 86%
- TypeScript unit tests: 100% statements
- TypeScript typecheck/lint/format
- golangci-lint
- markdownlint
- ProB model-check (`make check-machines`)
- Pi RPC smoke (`make smoke-pi`)
- Fake-pi integration test

**Constraints:**

- `make check` auto-formats then runs all automated gates.
- Coverage must not regress below 85% on core packages.
- New features require tests before or alongside implementation.

## ADR 13: golangci-lint matching ethos conventions

**Decision:** Use golangci-lint v2 with the same configuration as the
ethos project: govet, staticcheck, unused, gofmt. Pinned version in
Makefile and `.golangci.yml`.

**Evidence:** Adopted and passing. Local and CI run the same analyzer
versions.

**Constraints:**

- `make tools` installs the pinned version.
- `make format` runs `golangci-lint fmt`.
- `CGO_ENABLED=0` for builds; version injected via ldflags.
