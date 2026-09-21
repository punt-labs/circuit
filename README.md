# Circuit

[![Working Backwards](https://img.shields.io/badge/Working_Backwards-hypothesis-lightgrey)](./prfaq.pdf)

A tiny formal state-machine engine for agent workflow loops.

`circuit` is a small real project for testing whether agent harnesses such as
**pi**, **Claude Code**, and **opencode** can be guided by an explicit workflow
machine instead of a long prompt. The harness should know the current state,
which operations are valid from that state, and which facts must hold before the
workflow can progress.

Shipped workflow machines are authored as
[B-Method](https://en.wikipedia.org/wiki/B-Method) abstract machines, proven or
model-checked with ProB during development, and interpreted by Go at runtime
without requiring ProB for normal use.

Start with this README. The complete documentation index and naming convention
are in [`docs/README.md`](docs/README.md). Key references:

- [`docs/development.md`](docs/development.md) for the development environment;
- [`docs/testing.md`](docs/testing.md) for gates and test layers;
- [`docs/operations/run-contract.md`](docs/operations/run-contract.md) for
  Circuit-driven agent runs;
- [`docs/project/roadmap.md`](docs/project/roadmap.md) for planned work and its
  source documents;
- [`docs/architecture/decisions.md`](docs/architecture/decisions.md) for
  accepted architectural decisions;
- [`docs/architecture/b-machines.md`](docs/architecture/b-machines.md) for the
  original B-machine design rationale;
- [`docs/spec/circuit-runtime.tex`](docs/spec/circuit-runtime.tex) for the
  formal runtime design specification.

## Quick start

Enter the reproducible development shell, build the CLI, and inspect the
available machines:

```bash
nix develop
make build
./circuit list
```

Run a machine directly. Record the session ID printed by `start` and use it in
subsequent commands so the example remains unambiguous when other sessions
exist:

```bash
./circuit start build-job
./circuit status build-job-a3f8
./circuit advance start build-job-a3f8
./circuit advance finish build-job-a3f8
```

The final transition reaches a terminal state and stops that session. Session
state is persisted under `.tmp/sessions/`, so later CLI invocations resume it.
Replace the example ID with the value printed by `start`, then remove the stopped
example session before starting the driven example:

```bash
./circuit unload build-job-a3f8
```

To let Circuit drive a pi agent through a machine, install the checked-in pi
extension dependencies, then use a machine that has a companion prompt file:

```bash
npm --prefix .pi ci
./circuit drive tdd-flow --task "describe the implementation task"
```

This mode requires pi and model credentials. Circuit owns machine state and
transition checks; pi performs the work and requests transition events. See
[`docs/operations/run-contract.md`](docs/operations/run-contract.md) for the
prompt, check, response, and trace contracts.

## Why this exists

Punt Labs agent workflows are loops, not straight-line scripts. A pull request
watcher, for example, repeatedly observes GitHub state and branches based on
what it sees:

- CI failed or review findings appeared -> fix
- checks are green and review is clean -> merge
- nothing actionable yet -> wait and poll again

Today these loops are often retyped as prompts. `circuit` should make them
reviewed, versioned, executable contracts: named states, typed facts, guarded
operations, terminal states, and mechanically checked progress rules.

The central invariant is:

```text
A harness may request progress, but the machine decides whether progress is
valid.
```

## Current status

Implemented now:

- Nix development shell with golangci-lint matching ethos conventions
- `make check` gate for engine, RPC protocol, pi extension, and docs
- project-local pi extension at `.pi/extensions/circuit.ts` with context
  injection via `before_agent_start`, LLM tools matching the `/circuit` slash
  commands, and human control through those slash commands
- B machines: `build-job`, `pr-watch`, `review-flow`, `retry-flow`,
  `tdd-flow`
- Circuit-B multi-pass parser/evaluator under `internal/circuitb/`
- multi-session lifecycle runtime under `internal/circuitrun/` with machine-hex
  session IDs and auto-stop on terminal states
- RPC protocol logic under `internal/circuitrpc/` with fake-pi integration
  test
- CLI commands for machine discovery, validation, scaffolding, session
  lifecycle, transition requests, JSON output, and end-to-end agent driving
- check bindings for runtime preconditions with invocation tracking; machines
  with BOOL facts must load with complete bindings before they can start
- `retry-flow` machine proving block/retry loops work
- `tdd-flow` machine modeling red-green-refactor with external checks for
  observed failing tests and passing test suites
- separate implementation, structural-quality, formal-machine, formal-spec,
  and live pi smoke gates
- live evidence for pi-hosted tool use, blocked-transition retry, and
  Circuit-driven TDD/refactoring workflows

## Direction: B machines

Circuit workflow definitions should be B abstract machines.

A small machine looks like this:

```b
MACHINE BuildJob
SETS
    STATE = {idle, running, done};
    TRANSITION = {start, finish}
VARIABLES
    current
INVARIANT
    current : STATE
INITIALISATION
    current := idle
OPERATIONS
    Advance(evt) =
        PRE
            evt : TRANSITION &
            current /= done &
            (
                (current = idle & evt = start) or
                (current = running & evt = finish)
            )
        THEN
            IF current = idle & evt = start THEN
                current := running
            ELSIF current = running & evt = finish THEN
                current := done
            END
        END
END
```

This maps directly to the workflow problem:

| Circuit concept | B concept |
| --- | --- |
| workflow definition | `MACHINE` |
| states and operation names | enumerated `SETS` |
| current workflow position | `VARIABLES current` |
| observed facts | additional `VARIABLES` |
| safety constraints | `INVARIANT` |
| starting state | `INITIALISATION` |
| transition request | `OPERATION` |
| guard condition | `PRE` |
| state update | substitution after `THEN` |

Development can require ProB:

```bash
make check-machines
```

Runtime does not require ProB:

```bash
circuit list
circuit load review-flow
circuit scaffold review-flow
circuit start build-job
circuit status
circuit advance start
circuit advance finish

# With multiple sessions, target one explicitly:
circuit start build-job
circuit start review-flow
circuit status
circuit advance start build-job-a3f8
circuit stop review-flow-b4c9
```

The Go runtime parses and evaluates a strict Circuit-B profile. It does not try
to interpret all B. Valid B outside the profile fails with a clear runtime
diagnostic explaining which construct is unsupported.

## Runtime model

A circuit machine should answer four practical questions:

1. What machine is active?
2. What B-machine state is the active circuit in?
3. Which operations are enabled or blocked now?
4. If a requested operation is allowed, what is the next state?

The user-facing command for this is `status`, not `state`. State is the B-machine
variable. Status is the operational report about the active circuit. Today it
includes the active machine, current state, enabled operations, blocked
operations, and any check results collected during transition attempts. Later it
should also include runtime metadata such as start time, elapsed time, accepted
transition count, blocked transition count, and the latest accepted or blocked
operation.

Runtime preconditions that depend on the outside world are represented as B
booleans and bound to registered checks outside B. For example,
`review-flow.mch` requires `makeCheckPassed = TRUE` before advancing from
`coding` to `codeReview`; `review-flow.checks.yaml` binds that B variable to the
`makeCheck` registry entry in `check-registry.yaml`. `tdd-flow.mch` uses the
same pattern for red-green-refactor discipline: `writeTest` requires
`not(testSuitePassed = TRUE)` while `implement`, `reviewQuality`, and
`keepGreen` require `testSuitePassed = TRUE`. `finish` requires a separate
`codeQualityPassed = TRUE`, and `refactor` is available from `qualityReview`
when code quality is not passing. Circuit-B supports `not(...)` for these
negated gates, so no separate "failing test observed" fact is needed. A machine
with BOOL facts must load with complete bindings before it can start. Use
`circuit scaffold <machine>` to generate missing bindings and registry stubs;
stubs default to `false`, so incomplete integrations block safely.

Circuit manages multiple sessions. A session is one running instance of a
machine, identified as `<machine>-<4hex>` such as `build-job-a3f8`. Each session
has independent B state and check history. The runtime lifecycle is:

- `unloaded` — no session is selected or persisted
- `active` — machine is loaded in memory and internal workflow state is
  progressing
- `suspended` — machine is serialized to disk between short CLI invocations
- `stopped` — machine reached a terminal state or was explicitly stopped

Short-lived CLI commands implicitly resume and suspend:
`suspended → active → suspended`. When a machine reaches a terminal state
(no enabled operations), that session auto-stops without affecting other active
sessions. `status` reports known active or stopped sessions; with no known
session it reports "no session" instead of an error.

Active sessions are stored as JSON files under `.tmp/sessions/`. The old
`.tmp/circuit.suspended.json` path is read only for migration from the earlier
single-session runtime.

The harness adapter is responsible for UI and observation. The machine remains
the authority for valid progress.

For pi, that means two relationships are worth testing:

1. **Pi hosts the engine.** A pi extension calls the Go runtime, displays the
   current state, and exposes commands/tools to request valid operations.
2. **Circuit drives pi.** A circuit runner owns the machine state and uses pi RPC
   as an agent backend for observation and action.

Both relationships should use the same `.mch` file and the same Go evaluator.

## Circuit-B parser approach

The Go implementation treats Circuit-B as a small compiler problem, not as
string matching.

Current passes:

1. **Lex and parse structure.** Build a raw AST with source spans.
2. **Resolve names and types.** Distinguish variables, sets, enum values, and
   operation parameters.
3. **Validate the Circuit-B profile.** Reject unsupported B constructs with
   actionable diagnostics.
4. **Evaluate.** Compute enabled operations and apply supported substitutions.

Every token and AST node should retain source location information so that
syntax errors, type errors, and profile violations can point back to the author
source.

## Integrations

Circuit currently supports pi in two modes:

- **Pi-hosted:** project-local slash commands and LLM tools call the Go CLI;
  Circuit validates every requested transition.
- **Circuit-driven:** `circuit drive` owns the workflow and uses pi RPC as the
  agent backend.

Other harnesses can invoke the CLI in tool-call mode. Native Claude Code and
opencode integrations remain planned. Integration details and limitations are
in [`docs/operations/harnesses.md`](docs/operations/harnesses.md).

## Implementation status

The initial scaffold, toolchain, Circuit-B engine, session runtime, pi-hosted
adapter, and Circuit-driven pi path are implemented. The current work is product
validation and hardening rather than proving the basic architecture:

- compare guided and unguided outcomes on realistic workflows;
- broaden external evidence beyond local command checks;
- improve concurrent-session safety and selection UX;
- make machine authoring and handoff easier without weakening formal checks.

Current evidence and unresolved questions are tracked in
[`docs/project/risks.md`](docs/project/risks.md). Accepted architectural
decisions are recorded in
[`docs/architecture/decisions.md`](docs/architecture/decisions.md), not in
milestone prose here.

## Contributing

Contributor setup, development practices, quality gates, and pull-request
expectations are in [`CONTRIBUTING.md`](CONTRIBUTING.md).

## Design principle

Keep the runtime tiny and boring. Put the mathematical precision in the B
machine, prove it during development, and make harness adapters obey it at
runtime.
