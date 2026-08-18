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

**Profile expansions accepted:**

- `not(<predicate>)`: parenthesised boolean negation. Enables workflows that
  need to gate on the negation of an observed BOOL fact (for example, TDD `spec
  -> red` requires `not(testSuitePassed = TRUE)`).

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
- Check commands receive session-scoped environment variables such as
  `CIRCUIT_SESSION_ID`, `CIRCUIT_MACHINE_NAME`, `CIRCUIT_MACHINE_FILE`, and
  `CIRCUIT_CURRENT_STATE` so project-local evidence can stay isolated per
  concurrent machine session.

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
- The runner is intentionally single-session. Multi-session
  orchestration is supported in pi-hosted/CLI mode, not in this
  runner.

## ADR 10: LLM tools with full slash command parity

**Decision:** Every slash command has a corresponding LLM tool. The
agent can discover, load, scaffold, start, operate, stop, and unload
circuits without human intervention.

**Evidence:** Milestone 4 proved this. The agent called `circuit_status`
then `circuit_advance` unprompted and progressed `build-job` from
`idle` to `running`. All seven tools are registered; parsing,
formatting, and Go command behavior are tested.

**Constraints:**

- Tool names use underscores: `circuit_list`, `circuit_load`,
  `circuit_scaffold`, `circuit_start`, `circuit_status`,
  `circuit_advance`, `circuit_stop`, `circuit_unload`.
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

## ADR 14: Circuit-driven agent runs use machine-local state prompts

**Decision:** Circuit-driven agent runs combine the current B-machine
state with a machine-local prompt file at
`machines/<machine>.prompts.yaml`. The prompt file maps states to
state-specific work instructions and, optionally, the transition event
the agent should request when done.

**Evidence:** Plain context injection and generic "respond with an
event" prompts were insufficient for TDD. The agent needed to know what
`spec`, `red`, `green`, `qualityReview`, and `refactoring` meant in
operational terms. Hardcoding `tdd-flow` prompts in the CLI was tried
and rejected because it made Circuit know machine-specific semantics.
Moving prompts into `tdd-flow.prompts.yaml` preserved the generic driver
and made machine behavior data-driven.

**Constraints:**

- Circuit core must not hardcode per-machine prompt text.
- Prompt files are guidance, not truth. The B machine and checks remain
  the authority for progress.
- Missing or malformed prompt files are driver errors for `circuit drive`.
- Prompt files may be overridden in worktrees for experiments, but the
  driver must load them as data.

## ADR 15: Circuit drive is the enforcement path; Pi-hosted context is guidance

**Decision:** The enforcement model is `outer Pi/human -> Circuit ->
inner Pi`. `circuit drive <machine> --task <goal>` starts a machine,
loads machine prompts, drives `pi --mode rpc`, extracts a requested
event from the Pi response, runs Circuit checks, and advances only if
the machine accepts the transition.

**Evidence:** Pi-hosted Circuit context and tools helped the agent see
state but did not ensure the agent used the machine. The dogfood session
repeatedly required human reminders. The Circuit-driven path, tested
with `make smoke-drive`, showed Circuit prompting real Pi, accepting
valid events, rejecting blocked progress, and reaching terminal state.
Real `--json` slices were delivered through `tdd-flow` using this path.

**Rejected alternatives:**

- Relying on injected context alone. It informs but does not enforce.
- Requiring the driven Pi to call a `circuit_advance` tool directly.
  For the driver path, a plain event response is simpler: Pi requests;
  Circuit advances.
- Keeping the old `cmd/circuit-rpc-spike` as the primary interface. It
  proved feasibility but was not shaped for reusable guided runs.

**Constraints:**

- Pi may request progress but must not decide truth.
- Circuit owns state, check execution, and transition acceptance.
- The driver must support re-prompting when no event is extracted or a
  requested event is blocked.
- The agent backend is substitutable as long as it implements prompt ->
  response semantics.

## ADR 16: Driver traces are required for dogfood evaluation

**Decision:** `circuit drive` writes a JSONL trace to
`.tmp/circuit/<session-id>/drive.jsonl`. The trace records prompts,
responses, workspace status after each response, and accepted
transitions.

**Evidence:** Before tracing, a TDD smoke appeared to have produced an
invalid result, but the final diff was not enough to explain why. After
adding traces, we could see whether Pi edited tests in `spec`, edited
production code in `red`, chose `finish` or `refactor`, and which
transitions Circuit accepted. Tracing also exposed an environment issue:
a fresh worktree without `.pi/node_modules` made `make check` fail for
setup reasons and falsely satisfied the red gate.

**Constraints:**

- Trace is driver-level observability, not part of B-machine semantics.
- Trace files live under the session-scoped `.tmp/circuit/<session-id>/`
  directory.
- The trace must record enough information to diagnose process behavior
  without relying on memory of the run.
- Future work may type trace events more strongly; current JSONL is the
  minimal useful form.

## ADR 17: TDD finish is gated by code quality, not agent preference

**Decision:** `tdd-flow` separates `testSuitePassed` from
`codeQualityPassed`. The machine forces `green -> qualityReview` after
implementation. From `qualityReview`, `finish` requires
`codeQualityPassed = TRUE`; otherwise `refactor` is the valid path and
loops through `refactoring -> green -> qualityReview`.

**Evidence:** A prompt-only refactor suggestion did not reliably make Pi
choose refactoring. When code quality was made a formal BOOL gate,
Circuit forced repeated refactoring loops until `make check-go-quality`
passed. This produced the intended outcome: refactoring was no longer a
soft preference but the path required by failing quality checks.

**Rejected alternatives:**

- Letting `green` choose directly between `finish` and `refactor`. The
  agent tended to choose `finish`.
- Prompting "inspect for refactoring" without a quality gate. This made
  inspection visible but not enforceable.
- Treating `make check` as both behavior and quality. Test green and code
  quality green are different facts.

**Constraints:**

- `tdd-flow` references abstract BOOLs only: `testSuitePassed` and
  `codeQualityPassed`.
- Repositories substitute their own quality command through
  `check-registry.yaml`.
- `codeQualityPassed` must be useful enough to fail when refactoring is
  needed, but thresholds can be calibrated separately from the machine.

## ADR 18: Calibrated Go structural quality gate

**Decision:** The initial Go quality gate is
`make check-go-quality`, running golangci-lint with `dupl`, `gocognit`,
and `funlen` over production Go code. Thresholds are calibrated to
produce a small, actionable failure set rather than a flood:

```text
dupl threshold: 100
gocognit min-complexity: 20
funlen lines: 70
funlen statements: 40
```

**Evidence:** A very strict `dupl` threshold of 5 produced dozens of
low-signal one-line duplicates. A looser placeholder target produced no
signal. The calibrated thresholds produced a small set of failures per
category, and a real Circuit-driven refactor loop drove that set to zero.

**Rejected alternatives:**

- Leaving `check-go-quality` as `true`. It did not exercise
  `codeQualityPassed`.
- Adding many linters at once. It blurred the signal and made dogfood
  harder to interpret.
- Using extremely low duplication thresholds as the first enforced gate.
  It found too many syntactic coincidences before the refactoring loop was
  ready.

**Constraints:**

- The gate is a starting point, not final policy.
- It excludes tests for now to focus on production structure.
- Future tools or stricter thresholds should be introduced through TDD
  slices and observed through `tdd-flow`.

## ADR 19: CI runs the full formal and implementation gates through Nix

**Decision:** GitHub Actions runs `make check`, `make check-go-quality`, and
`make check-machines` inside the pinned Nix dev shell for pushes to `main` and
`feat/**`, and for pull requests targeting `main`. ProB 1.15.1 is packaged by a
project-local Nix derivation and is a first-class CI dependency.

**Evidence:** Before CI existed, `pr-watch` required a manual
`SKIP_CI_CHECK=true` override and could not validate pull-request checks. The
first CI rollout caught platform-specific ProB dependencies that passed on
macOS but failed on clean Linux. Repeated CI cycles identified the full runtime
set: libstdc++, libgcc, libuuid, GMP, and a JRE for the Java parser. Once these
were included, the complete B-machine gate passed on GitHub Actions.

**Constraints:**

- ProB is a development/release dependency, never a circuit runtime dependency.
- ProB is fetched only from the official HHU release host and pinned by version
  and SHA-256.
- Linux ProB is x86_64-only. The flake keeps an aarch64 Linux Go-only dev shell
  but does not expose the ProB package there.
- CI actions are pinned by commit SHA and use read-only repository permissions.
- CI uses `npm ci`, not `npm install`, because `.pi/package-lock.json` is
  committed.
