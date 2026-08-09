# circuit

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

Design note: [`docs/design/b-machines.md`](docs/design/b-machines.md).

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

- Nix development shell
- `make check` gate for engine, pi extension, and docs
- project-local pi extension at `.pi/extensions/circuit.ts`
- TypeScript typecheck/lint/format gate for the pi extension
- B-machine spikes under `machines/build-job.mch`, `machines/pr-watch.mch`, and
  `machines/review-flow.mch`
- Circuit-B runtime package under `internal/circuitb/`
- Go runtime commands for the B machine:
  - `circuit list`
  - `circuit start build-job`
  - `circuit status`
  - `circuit advance start`
  - `circuit advance finish`
- check binding files for runtime preconditions:
  - `machines/review-flow.checks.yaml`
  - `machines/check-registry.yaml`
- ProB development gate: `make check-machines`

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
circuit start build-job
circuit status
circuit advance start
circuit advance finish
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
`makeCheck` registry entry in `check-registry.yaml`.

Circuit runtime is in-memory first. Short-lived CLI commands implicitly resume a
suspended runtime, operate in memory, then suspend again to
`.tmp/circuit.suspended.json`. That file is a pause/resume artifact, not the
conceptual source of state. A future long-running runtime should use the same
model and suspend only on exit or explicit pause.

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

## Nix-first development

`circuit` is Nix-first from the start.

The dev shell provides Go, Node, markdown linting, staticcheck, Beads, GitHub
CLI, and shell tooling. Normal development should happen inside the Nix shell:

```bash
nix develop
make check
```

Non-Nix development should remain possible for contributors who already have
the required tools installed.

## Make targets

The root Makefile is organized around product surfaces, not implementation
languages:

- `check-engine` validates the Go engine/CLI.
- `check-pi-extension` validates the project-local pi extension.
- `check-docs` validates Markdown documentation.
- `check-machines` validates B machines with ProB for development/release.
- `check` runs the automated aggregate gate.

Compatibility aliases remain:

- `lint` -> `lint-engine`
- `test` -> `test-engine`
- `build` -> `build-engine`
- `docs` -> `check-docs`

## Harness testbed

See [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) for local setup,
[`docs/HARNESS.md`](docs/HARNESS.md) for harness notes, and
[`docs/TESTING.md`](docs/TESTING.md) for the testing pyramid.

`circuit` is also our smallest cross-harness test project. We will test each
harness according to its own idioms instead of forcing Claude Code conventions
onto the others.

### Shared instructions

`AGENTS.md` is the intended harness-neutral instruction source. Claude Code may
use `CLAUDE.md` as a small entrypoint/addendum. Pi's observed behavior in this
repo loads the shared instructions from `AGENTS.md`.

### pi

Pi support currently includes a project-local TypeScript extension:

```text
.pi/extensions/circuit.ts
```

The extension shells out to the Go CLI and keeps transition logic in Go. Current
commands:

- `/circuit list`
- `/circuit start <machine>`
- `/circuit status`
- `/circuit advance <event>`
- `/circuit stop`

The pi-hosted spike is intentionally thin: pi presents circuit status and
requests transitions, while the B-backed Go runtime decides whether progress is
valid. The extension tracks one active circuit in the pi extension process after
`/circuit start <machine>`.

The other relationship — circuit as the outer runner driving `pi --mode rpc` —
was tested in spike 3 (`cmd/circuit-rpc-spike/`). The runner owned the B-machine
state, sent prompts, observed `agent_settled`, extracted the agent's chosen
operation, and validated it against the machine before advancing.

### Claude Code

Claude Code support should stay native:

- `CLAUDE.md` for Claude-specific entrypoint/addendum
- optional `.claude/commands/` later
- no hooks or subagents until the B-machine contract is stable

### opencode

Opencode support should stay native:

- `AGENTS.md`
- optional `opencode.json` once we actively test opencode

## Near-term milestones

### Milestone 0: scaffold

Done:

- README
- Nix-first development decision
- Go language decision
- harness strategy

### Milestone 1: toolchain gates

Done:

- `flake.nix`
- `flake.lock`
- `make check`
- Go/staticcheck/markdownlint tooling
- pi extension TypeScript tooling

### Milestone 2: B-machine foundation

Done:

- `machines/build-job.mch`, `machines/pr-watch.mch`, `machines/review-flow.mch`
- ProB development gate: `make check-machines`
- multi-pass Circuit-B lexer/parser/evaluator in Go (`internal/circuitb`)
- suspend/resume runtime (`internal/circuitrun`)
- CLI: `list`, `start`, `status`, `advance`, `stop`
- check bindings: `review-flow.checks.yaml`, `check-registry.yaml`
- golangci-lint adopted matching ethos conventions
- test coverage ≥85% on core packages

### Milestone 3: harness spikes

Done:

- spike 2: pi hosts circuit — `/circuit` commands shell out to Go CLI
- spike 3: circuit drives pi — Go runner owns B-machine state, sends prompts
  to `pi --mode rpc`, validates agent responses against the machine
- both relationships use the same `.mch` files and Go evaluator

### Milestone 4: usefulness proof

Next:

- context injection: agent sees state and valid operations automatically
- LLM tools: agent calls `circuit_advance`, not human `/circuit advance`
- gating: agent cannot claim progress the machine has not validated
- real workflow machine with facts and external check bindings

## Design principle

Keep the runtime tiny and boring. Put the mathematical precision in the B
machine, prove it during development, and make harness adapters obey it at
runtime.
