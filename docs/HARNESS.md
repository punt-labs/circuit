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

- `/circuit machine list` lists available machines from `machines/*.mch`.
- `/circuit machine check <machine>` validates a machine and its check bindings without
  starting a session.
- `/circuit machine scaffold <machine>` generates missing BOOL check bindings and
  failing registry stubs.
- `/circuit session start <machine>` starts a new circuit session from a named machine.
- `/circuit session list` shows all known active or stopped sessions.
- `/circuit session show <session>` shows the selected session.
- `/circuit session advance <event> [session]` requests `Advance(event)` against the
  only active session or a selected session.
- `/circuit session stop [session]` stops the only active session or a selected session.
- `/circuit session remove <session>` removes a stopped session from runtime storage.
- `/circuit status` summarizes workspace session counts for humans.

LLM tools (machine and session operations):

- `circuit_machine_list` lists available machines.
- `circuit_machine_check` validates a machine and its check bindings.
- `circuit_machine_scaffold` generates missing BOOL check bindings and failing registry
  stubs.
- `circuit_session_start` starts an active circuit from a named machine.
- `circuit_session_list` lists all known sessions and their operations.
- `circuit_session_show` reports the state and valid operations of one session;
  it requires a session ID.
- `circuit_session_advance` requests a transition; the B machine validates the
  precondition and returns allowed or blocked. It accepts an optional session ID
  when multiple sessions are active.
- `circuit_session_stop` stops an active circuit session; it accepts an optional
  session ID. Stopping a known stopped session is idempotent.
- `circuit_session_remove` removes a stopped session from runtime storage.

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
/circuit machine list
/circuit machine check build-job
/circuit session start build-job
/circuit session list
/circuit session advance start
/circuit session list
/circuit session advance finish
```

The start command should show a session ID, `current: idle`, `Advance(start)`
enabled, and `Advance(finish)` blocked. After `/circuit session advance start`,
`/circuit session list` should show `current: running` with `Advance(finish)` enabled.
After `/circuit session advance finish`, that session reaches `done` and auto-stops.
With more than one active session, use `/circuit session advance <event> <session>` and
`/circuit session stop <session>`.

The precondition smoke uses `review-flow`:

```text
/circuit session start review-flow
/circuit session advance requestReview
/circuit session list
```

`requestReview` runs the registered `makeCheck` command, stores the boolean in
`makeCheckPassed`, increments its invocation count, and then lets the B machine
accept or block the transition.

## Circuit drives pi

The other relationship — circuit as the outer runner driving `pi --mode rpc` —
was first tested in spike 3 (`cmd/circuit-rpc-spike/`) and now ships as
`circuit session drive <machine> [--task <goal>]` (see DESIGN.md ADR 14–16). The runner
owns the B-machine state, sends the state prompt from
`machines/<machine>.prompts.yaml`, waits for pi's `agent_settled` event,
extracts the agent's chosen operation, and validates it against the machine
before advancing. `make smoke-drive` exercises this path against a live pi.

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
