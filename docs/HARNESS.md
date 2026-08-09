# Harness Notes

`circuit` is Punt Labs' smallest cross-harness project. The product is a small
state-machine engine; the repo also lets us compare how pi, Claude
Code, and opencode consume project instructions.

## Instruction files

The intended shared instruction source is:

```text
AGENTS.md
```

Harness-specific entrypoints may reference or layer on top of it.

## Claude Code

Claude Code uses:

```text
CLAUDE.md
```

For `circuit`, `CLAUDE.md` is intentionally a small stub that imports
`AGENTS.md` and reserves space for future Claude Code-specific notes.

We are not adding Claude Code hooks, subagents, or custom commands in the
initial scaffold.

## pi

Pi documentation says pi loads context files from global, parent, and current
directories, including:

```text
AGENTS.md
CLAUDE.md
AGENTS.override.md
```

Pi project resources under `.pi/` require project trust. Context files are loaded
unless context discovery is disabled.

For `circuit`, pi support starts minimal:

- use the Nix dev shell
- use `AGENTS.md` / `CLAUDE.md` context behavior as observed
- use tmux for visible long-running terminal monitoring
- load a small project-local extension from `.pi/extensions/circuit.ts`
- expose thin commands that wrap the Go CLI and do not duplicate transition
  logic

Current extension commands:

Slash commands (human control):

- `/circuit list` lists available machines from `machines/*.mch`.
- `/circuit start <machine>` starts an active circuit from a named machine.
- `/circuit status` shows active circuit status.
- `/circuit advance <event>` requests `Advance(event)` against the active
  circuit.
- `/circuit stop` clears the active circuit.

LLM tools (agent calls these automatically):

- `circuit_status` reports active circuit state and valid operations.
- `circuit_advance` requests a transition; the B machine validates the
  precondition and returns allowed or blocked.

Context injection:

- On every agent turn, `before_agent_start` injects the current circuit
  state and valid operations into the agent's context so the agent sees
  the machine without manual commands.

## opencode

Opencode supports `AGENTS.md` and can also use `opencode.json` for project
configuration. Do not add `opencode.json` until we actively test opencode in this
repo.

## Observed pi behavior

A pi startup check from the `circuit` repo showed these loaded context files:

```text
~/Coding/punt-labs/CLAUDE.md
~/Coding/punt-labs/circuit/AGENTS.md
```

With both `AGENTS.md` and `CLAUDE.md` present in `circuit`, pi loaded
`AGENTS.md` for the repo-local context and did not list the repo-local
`CLAUDE.md`. This supports the intended split: shared repo instructions in
`AGENTS.md`, Claude Code entrypoint/addendum in `CLAUDE.md`.

The same startup check warned that tmux is using `extended-keys-format xterm`;
pi recommends `csi-u` for best modified-key handling.

A later pi startup check with project approval showed the project extension
loaded from `.pi/extensions/circuit.ts`. Running the extension command against

The pi-hosted Circuit spike keeps pi as the interactive harness and delegates
machine semantics to Go. The expected manual smoke path is:

```text
/circuit list
/circuit start build-job
/circuit status
/circuit advance start
/circuit status
/circuit advance finish
```

The start command should show `current: idle`, `Advance(start)` enabled, and
`Advance(finish)` blocked. After `/circuit advance start`, `/circuit status`
should show `current: running` with `Advance(finish)` enabled. After
`/circuit advance finish`, the active circuit reaches `done`.

The precondition smoke uses `review-flow`:

```text
/circuit start review-flow
/circuit advance requestReview
/circuit status
```

`requestReview` runs the registered `makeCheck` command, stores the boolean in
`makeCheckPassed`, increments its invocation count, and then lets the B machine
accept or block the transition.

## Open questions

- Is the Claude Code `@AGENTS.md` stub enough, or do we need a different split?
- Does opencode need explicit `opencode.json` instructions, or is `AGENTS.md`
  sufficient?
- Should `AGENTS.override.md` be used for pi if shared instructions become too
  Claude-specific?

## Current policy

Do not centralize or render harness configuration yet. First, observe each
harness' native behavior in this small repo.
