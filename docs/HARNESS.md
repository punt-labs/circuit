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
- `/circuit load <machine>` validates a machine and its check bindings without
  starting a session.
- `/circuit scaffold <machine>` generates missing BOOL check bindings and
  failing registry stubs.
- `/circuit start <machine>` starts a new circuit session from a named machine.
- `/circuit status [session]` shows all known active or stopped sessions, or one
  selected session.
- `/circuit advance <event> [session]` requests `Advance(event)` against the
  only active session or a selected session.
- `/circuit stop [session]` stops the only active session or a selected session.
- `/circuit unload <session>` removes a stopped session from runtime storage.

LLM tools (full parity with slash commands):

- `circuit_list` lists available machines.
- `circuit_load` validates a machine and its check bindings.
- `circuit_scaffold` generates missing BOOL check bindings and failing registry
  stubs.
- `circuit_start` starts an active circuit from a named machine.
- `circuit_status` reports known session state and valid operations; it accepts
  an optional session ID.
- `circuit_advance` requests a transition; the B machine validates the
  precondition and returns allowed or blocked. It accepts an optional session ID
  when multiple sessions are active.
- `circuit_stop` stops an active circuit session; it accepts an optional
  session ID. Stopping a known stopped session is idempotent.
- `circuit_unload` removes a stopped session from runtime storage.

Context injection:

- On every agent turn, `before_agent_start` injects all active circuit
  sessions, their current states, and valid operations into the agent's context.
  No injection happens when no session is active or for sessions after
  terminal auto-stop.

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
loaded from `.pi/extensions/circuit.ts`.

The pi-hosted Circuit spike keeps pi as the interactive harness and delegates
machine semantics to Go. The expected manual smoke path is:

```text
/circuit list
/circuit load build-job
/circuit start build-job
/circuit status
/circuit advance start
/circuit status
/circuit advance finish
```

The start command should show a session ID, `current: idle`, `Advance(start)`
enabled, and `Advance(finish)` blocked. After `/circuit advance start`,
`/circuit status` should show `current: running` with `Advance(finish)` enabled.
After `/circuit advance finish`, that session reaches `done` and auto-stops.
With more than one active session, use `/circuit advance <event> <session>` and
`/circuit stop <session>`.

The precondition smoke uses `review-flow`:

```text
/circuit start review-flow
/circuit advance requestReview
/circuit status
```

`requestReview` runs the registered `makeCheck` command, stores the boolean in
`makeCheckPassed`, increments its invocation count, and then lets the B machine
accept or block the transition.

## Circuit drives pi

The other relationship — circuit as the outer runner driving `pi --mode rpc` —
was tested in spike 3 (`cmd/circuit-rpc-spike/`). The runner owned the B-machine
state, sent prompts, observed `agent_settled`, extracted the agent's chosen
operation, and validated it against the machine before advancing.

The circuit-drives-pi runner is intentionally single-session. Multi-session
orchestration is supported in pi-hosted/CLI mode, not in this runner.

## Open questions

- Is the Claude Code `@AGENTS.md` stub enough, or do we need a different split?
- Does opencode need explicit `opencode.json` instructions, or is `AGENTS.md`
  sufficient?
- Should `AGENTS.override.md` be used for pi if shared instructions become too
  Claude-specific?

## Current policy

Do not centralize or render harness configuration yet. First, observe each
harness' native behavior in this small repo.
