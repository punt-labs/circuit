# circuit

A tiny state-machine engine for agent workflow loops.

`circuit` starts as a deliberately small real project: implement the smallest
useful subset of Punt Labs' playbook state-machine design, while dogfooding our
development tooling across **pi**, **Claude Code**, and **opencode**. The core
product is the state-machine engine; harness support is how we test and use it,
not the product itself.

## Why this exists

Punt Labs agent workflows are loops, not straight-line scripts. A pull request
watcher, for example, repeatedly observes GitHub state and branches based on
what it sees:

- CI failed or review findings appeared -> fix
- checks are green and review is clean -> merge
- nothing actionable yet -> wait and poll again

Today these loops are often retyped as prompts. `circuit` makes them reviewed,
versioned artifacts: named states, guarded transitions, polling ticks, and
terminal states.

The design seed is Punt Kit's playbook state-machine draft:

```text
punt-kit/docs/designs/playbook-state-machines.md
```

## Initial scope

The first useful command should be:

```bash
circuit validate examples/pr-watch.yaml
```

The initial engine validates structure only. It does **not** execute real PR
workflows yet.

Planned first subset:

- parse YAML playbooks
- require exactly one of `steps` or `states`
- for state machines, use the first state as the initial state
- validate unique state IDs
- validate transition targets
- detect terminal states
- detect stuck non-terminal states without `poll`
- print clear diagnostics
- print a small human-readable summary

Out of scope for the first milestone:

- GitHub API integration
- scheduler daemon
- persistent run-state
- MCP server
- config renderer
- permission/policy generator
- full playbook execution

## Language

`circuit` is a Go project.

Reasons:

- small, fast CLI
- explicit structs for schema validation
- easy static-ish binaries and `go install`
- good fit for state-machine graph checks
- thin harness adapters can shell out to the CLI
- Punt Labs already has Go tooling and standards from projects like Beadle and
  Ethos

The pi extension, when added, will be TypeScript because pi extensions are
TypeScript. It should remain a wrapper around the Go CLI, not a second engine.

## Nix-first development

`circuit` is Nix-first from the start.

Nix will provide system/toolchain dependencies such as:

- Go 1.26
- `gopls`
- `go-tools` / `staticcheck`
- `markdownlint-cli2`
- shell tooling (`git`, `gh`, `jq`, `ripgrep`, `shellcheck`, etc.)

Go modules still own Go dependencies. Nix owns the toolchain, not vendored Go
packages.

Expected workflow once `flake.nix` exists:

```bash
nix develop
make check
```

Non-Nix development should remain possible:

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
make check
```

Nix is a supported reproducible dev environment, not a required runtime.

## Harness testbed

`circuit` is also our smallest cross-harness test project. We will test each
harness according to its own idioms instead of forcing Claude Code conventions
onto the others.

### Shared instructions

We will test whether `AGENTS.md` can be the harness-neutral instruction source.
Claude Code may use a small `CLAUDE.md` stub/addendum if needed. Pi's documented
context loading includes `AGENTS.md`, `CLAUDE.md`, and `AGENTS.override.md`; we
will verify precedence empirically in this repo before standardizing a pattern.

### pi

Pi support should start minimal:

- use normal context files (`AGENTS.md` / `CLAUDE.md` behavior)
- use Nix for tools
- use `tmux` + `watch` for visible PR monitoring
- add a project-local pi extension only after the CLI exists

First pi extension idea:

```text
/circuit validate examples/pr-watch.yaml
```

The extension should shell out to the Go CLI.

### Claude Code

Claude Code support should stay native:

- `CLAUDE.md` for Claude-specific entrypoint/addendum
- optional `.claude/commands/` later
- no hooks or subagents in the initial scaffold

### opencode

Opencode support should stay native:

- `AGENTS.md`
- optional `opencode.json` once we actively test opencode

## Proposed repository shape

Initial minimal shape:

```text
README.md
AGENTS.md
CLAUDE.md
Makefile
flake.nix
flake.lock
go.mod
cmd/circuit/main.go
internal/playbook/
examples/pr-watch.yaml
docs/DEVELOPMENT.md
docs/HARNESS.md
```

Do not add all of this at once unless a milestone needs it. The first commit can
remain mostly documentation and scaffold.

## Milestones

### Milestone 0: scaffold

- README
- Nix-first development decision
- Go language decision
- harness strategy

### Milestone 1: minimal CLI

- `circuit validate <file>`
- `circuit summary <file>`
- YAML parsing
- structural validation
- tests

### Milestone 2: Nix dev shell

- `flake.nix`
- `flake.lock`
- `make check`
- Go/staticcheck/markdownlint tooling

### Milestone 3: pi extension

- `.pi/extensions/circuit.ts`
- register a small command
- shell out to `circuit validate`
- document trust/reload behavior

### Milestone 4: opencode and Claude Code comparison

- verify instruction loading
- add minimal native config if useful
- document tradeoffs

## Design principle

Keep the engine tiny and boring. Use this repo to expose tooling friction, not
to hide it behind abstractions too early.
