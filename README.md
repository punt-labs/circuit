# circuit

> A small formal state-machine engine for agent workflow loops.

[![License](https://img.shields.io/github/license/punt-labs/circuit)](LICENSE)
[![CI](https://img.shields.io/github/actions/workflow/status/punt-labs/circuit/check.yml?label=CI)](https://github.com/punt-labs/circuit/actions/workflows/check.yml)
[![Working Backwards](https://img.shields.io/badge/Working_Backwards-hypothesis-lightgrey)](./prfaq.pdf)

`circuit` runs workflow state machines that say which steps are allowed next
and refuse a step whose conditions are not met. It is for people who run coding
agents in the pi coding agent and want a workflow, such as test-driven
development or a pull request review loop, enforced by a checked model instead
of a long prompt. Each workflow is a [B-Method](https://en.wikipedia.org/wiki/B-Method)
abstract machine (a `.mch` file); the machines are model-checked with
[ProB](https://prob.hhu.de/) during development, and a Go binary runs them
without ProB. The project is at hypothesis stage: the repository is private and
has no releases or packaged installer.

**Platforms:** macOS, Linux

## Quick Start

There is no packaged installer yet, so build and install from source. You need
read access to the private `punt-labs/circuit` repository, git, make, and Go
(the minimum version is in `go.mod`).

```bash
git clone https://github.com/punt-labs/circuit.git
```

```bash
cd circuit
```

```bash
make install
```

`make install` builds the `circuit` binary and copies it to `~/.local/bin`.
Make sure that directory is on your `PATH`.

`circuit` reads machines from the `machines/` directory under the current
directory, so run it from the root of the checkout:

```bash
circuit machine list
```

### Build Without Installing

Build the binary without installing it. The binary is written to `./circuit`:

```bash
make build
```

Run `./circuit install` to install that binary to `~/.local/bin`, or copy it
to another directory on your `PATH`.

### Check the Install

Run `circuit doctor` to check local installation health. From the root of
the checkout, start the `build-job` example machine:

```bash
circuit session start build-job
```

The output shows `current: idle`, with `Advance(start)` enabled and
`Advance(finish)` blocked. Request the two events in order:

```bash
circuit session advance start
```

```bash
circuit session advance finish
```

The commands print `advanced: idle -> running` and then
`advanced: running -> done`. The session stops on its own when it reaches
`done`.

## Features

- **B machines as workflows** --- each workflow is a B abstract machine that
  ProB can model-check and the Go binary can run.
- **Enforced transitions** --- `circuit session advance` applies an event only when the
  machine's precondition holds; otherwise it prints `blocked:` and each failing
  condition, and the state does not change.
- **External checks** --- a machine's boolean variables can be bound to shell
  commands; `circuit` runs each command when an event is requested and uses its
  exit status as the value.
- **Multiple sessions** --- several machines can run at once, each with its own
  session ID.
- **Automatic stop** --- a session stops when it reaches a state with no
  enabled events.
- **JSON output** --- `--json` returns one structured value on every
  result-bearing command.
- **pi extension** --- `/circuit` commands and `circuit_*` tools inside pi, with
  the state of each active session added to the agent's context on every turn.
- **Agent-driven runs** --- `circuit session drive` runs a machine from start to finish
  with pi as the agent and advances only on events the machine accepts.

## What It Looks Like

A session of the `build-job` machine, including a blocked request:

```text
$ circuit session start build-job
started: build-job
session: build-job-4156
session: build-job-4156
session-state: active
machine: build-job
current: idle
enabled:
  Advance(start)
blocked:
  Advance(finish)
$ circuit session advance finish
blocked: Advance(finish)
  needs: evt = start
$ circuit session advance start
advanced: idle -> running
$ circuit session advance finish
advanced: running -> done
```

A machine with an external check. The `retry-flow` check fails on its first
run and passes on its second:

```text
$ circuit session advance proceed
blocked: Advance(proceed)
  needs: gateOpen = TRUE
checks:
  gateOpen: FALSE (invocations: 1)
$ circuit session advance proceed
advanced: waiting -> done
checks:
  gateOpen: TRUE (invocations: 2)
```

## Commands

| Command | What exit zero proves |
|---------|-------------|
| `circuit machine list` | Enumerated machine names from `machines/`. |
| `circuit machine check <machine>` | Validated the machine and its bindings without starting a session. |
| `circuit machine scaffold <machine>` | Generated any missing bindings and registry stubs that return false. |
| `circuit session start <machine>` | Started and persisted a new session and returned its details. |
| `circuit session list` | Read all active and stopped session reports. |
| `circuit session show <session>` | Read the report for the named session. |
| `circuit session advance <event> [session]` | Evaluated the event and persisted the outcome. A blocked outcome (`allowed: false` in JSON) leaves the workflow state unchanged; nothing is queued. Name the session when more than one is active. |
| `circuit session stop [session]` | The selected session is stopped. Name the session when more than one is active. |
| `circuit session remove <session>` | Removed the stopped session. |
| `circuit session drive <machine> [--task <goal>]` | Reached a terminal state with Pi and persisted the session. See [Driving pi](#driving-pi). |
| `circuit version` | Printed the binary version. |
| `circuit status` | Read this workspace's active and stopped session counts. |
| `circuit doctor` | All required installation checks passed. Git and Pi checks are optional. |
| `circuit install` | Installed the running binary to `~/.local/bin/circuit`. |
| `circuit completion bash` | Printed a Bash completion script. |

Run `circuit --help` or `circuit` for the tree, and add `--help` (or `-h`)
at any path for that command's output and exit contract. Pass `--json` before
or after a command path for one JSON value: an array for a list, an object
for a detail or action, or an object with an `error` string on failure.
`--verbose` (`-v`) writes outcome diagnostics to stderr; `--quiet` (`-q`)
suppresses progress, not results. They cannot be combined.

Completion writes a script, not JSON. To enable it in Bash for the current
shell (or add the line to `~/.bashrc`):

```bash
source <(circuit completion bash)
```

A session is one running instance of a machine. Its ID is the machine name plus
four hex digits, such as `build-job-a3f8`.

### Session storage

Circuit saves session JSON in `sessions/` and drive traces in `traces/` under
the private state directory:

| Workspace | Private state directory |
|-----------|-------------------------|
| Git checkout | `.punt-labs/local/circuit/` at the checkout root |
| Linked Git worktree | `.punt-labs/local/circuit/worktrees/<key>/` in the main checkout or, for a bare or separate Git directory, in the common Git directory |
| Directory outside Git | `~/.punt-labs/circuit/data/workspaces/<key>/` |

Invoking Circuit from a subdirectory of a checkout keeps separate sessions in
`workspaces/<key>/` below that checkout's private state directory. A key is
the SHA-256 of the physical absolute workspace directory (the checkout root
for the worktree key), with no trailing newline. Machine checks still run from
the directory where you invoke Circuit. Circuit creates a private ignore file
when writing Git workspace state, so routine Git operations do not pick it up.
Publishing that ignore file requires a filesystem that supports hard links.
For linked worktrees whose common Git directory is named `.git`, Circuit
uses Git on `PATH` to distinguish bare repositories from regular checkouts.
Moving a keyed directory changes its key; copy the old state directory to
the new key if you need to keep those sessions.

Circuit does not read or remove the old `.tmp` store. Before running the
new version, you can copy `.tmp/sessions/<id>.json` to `sessions/<id>.json`
and `.tmp/circuit/<id>/drive.jsonl` to `traces/<id>/drive.jsonl` under the
new private state directory. Copy an older `.tmp/circuit.suspended.json`
to `circuit.suspended.json` there if you still need to resume it. Copy
only into unoccupied destinations, and keep paths referenced by saved
sessions accessible. Leave check evidence at the locations expected by your
checks; it is not Circuit-managed state.

### Diagnostics and telemetry

Circuit writes private, rotating diagnostics to
`~/.punt-labs/circuit/logs/circuit.log`. By default it also records the command
name, outcome, and elapsed time in the local `~/.punt-labs/circuit/telemetry.db`
for 90 days. Neither store records command arguments or session and machine
content, and Circuit does not send telemetry over the network.

To disable telemetry, set `PUNT_TELEMETRY=off` or `CIRCUIT_TELEMETRY=off`, or
create `~/.punt-labs/circuit/telemetry-off`. Any of these switches stops new
capture and pauses retention cleanup without deleting existing data or
disabling diagnostics.

## Setup

### Writing a Machine

Add a `<name>.mch` file to `machines/`. `circuit` runs a restricted subset of B
called the Circuit-B profile. A machine outside the profile fails to load with
an error that points to the unsupported part. The smallest shipped machine is
`machines/build-job.mch`:

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

How the parts of a B machine map to a workflow:

| Workflow concept | B concept |
| --- | --- |
| workflow definition | `MACHINE` |
| states and event names | enumerated `SETS` |
| current workflow position | `VARIABLES current` |
| observed facts | additional `VARIABLES` |
| safety constraints | `INVARIANT` |
| starting state | `INITIALISATION` |
| event request | `OPERATION` |
| condition for the event | `PRE` |
| state update | substitution after `THEN` |

The full profile is in [B Machines for Circuit](docs/design/b-machines.md).

### Check Bindings

A check binding connects a `BOOL` variable in a machine to a shell command. Two
files in `machines/` define it:

- `<machine>.checks.yaml` maps each variable to a registry entry.
- `check-registry.yaml` defines each registry entry as a command that returns
  `BOOL`.

For example, `review-flow` binds `makeCheckPassed` to `make check`:

```yaml
# machines/review-flow.checks.yaml
checks:
  makeCheckPassed:
    use: makeCheck
```

```yaml
# machines/check-registry.yaml
checks:
  makeCheck:
    kind: command
    command: make check
    returns: BOOL
```

`circuit` runs the command with `sh -c` in the directory where `circuit` runs.
Exit status 0 means `TRUE`; any other status means `FALSE`. The command
receives `CIRCUIT_SESSION_ID`, `CIRCUIT_MACHINE_NAME`, `CIRCUIT_MACHINE_FILE`
and `CIRCUIT_CURRENT_STATE` in its environment.

A machine with `BOOL` variables cannot start until every variable has a
binding. `circuit machine scaffold <machine>` adds the missing bindings and registry
entries. Each generated entry runs `false`, so the event it guards stays
blocked until you replace the command.

## Machines

The repository ships five example machines:

| Machine | States | What it models |
|---------|--------|----------------|
| `build-job` | `idle`, `running`, `done` | The smallest workflow; no checks |
| `retry-flow` | `waiting`, `done` | A blocked request followed by a retry; its check (`machines/alternating-check.sh`) fails on odd runs and passes on even runs |
| `review-flow` | `coding`, `codeReview`, `done` | Code review that may start only after `make check` passes |
| `tdd-flow` | `spec`, `red`, `green`, `qualityReview`, `refactoring`, `done` | Test-driven development: writing a test requires a failing `make check`, implementing requires it to pass, and finishing also requires `make check-go-quality` to pass |
| `pr-watch` | `watch`, `fixing`, `done` | Fixing a pull request: after `needsWork`, the `fixed` event is accepted only when CI checks are green and no review thread is unresolved |

The `review-flow` and `tdd-flow` checks run this repository's own quality
gates, so they need its development tools (see [Contributing](CONTRIBUTING.md)).
The `pr-watch` checks use the GitHub CLI (`gh`) and read the pull request
number and repository from `.tmp/circuit/<session>/pr-watch.env`, which must
set `PR_NUMBER` and `REPO`.

## pi Extension

The repository includes a project-local pi extension,
`.pi/extensions/circuit.ts`. pi loads it when you start pi in the root of the
checkout and approve the project's resources. The extension runs
`go run ./cmd/circuit`, so it needs Go and the source checkout; it does not use
the installed binary.

| Slash command | Tool | What it does |
|---------------|------|--------------|
| `/circuit machine list` | `circuit_machine_list` | List machines |
| `/circuit machine check <machine>` | `circuit_machine_check` | Validate a machine and its check bindings |
| `/circuit machine scaffold <machine>` | `circuit_machine_scaffold` | Add missing check bindings and stubs |
| `/circuit session start <machine>` | `circuit_session_start` | Start a session |
| `/circuit session list` | `circuit_session_list` | List all known sessions |
| `/circuit session show <session>` | `circuit_session_show` | Show one session |
| `/circuit status` | — | Summarize workspace session counts |
| `/circuit session advance <event> [session]` | `circuit_session_advance` | Request an event |
| `/circuit session stop [session]` | `circuit_session_stop` | Stop a session |
| `/circuit session remove <session>` | `circuit_session_remove` | Remove a stopped session |

You type the slash commands; the agent calls the tools. Both run the same
`circuit` command, which decides whether an event is accepted. While at least
one session is active, the extension adds each active session's current state
and enabled events to the agent's context at the start of every turn.

## Driving pi

`circuit session drive <machine> --task "<goal>"` reverses the relationship: `circuit`
runs pi as the agent. It starts a session, launches `pi --mode rpc`, and for
each state sends the prompt from `machines/<machine>.prompts.yaml` together with
the goal. It reads the event pi asks for, runs the machine's checks, and
advances only if the machine accepts the event. The run ends when the session
reaches a terminal state.

- pi must be on your `PATH` and configured with a model.
- The machine needs a prompts file. `build-job`, `tdd-flow` and `pr-watch` have
  one.
- pi works in the current directory and can change files there.
- Each run writes a JSON Lines trace of prompts, responses, `git status` output
  and transitions to `traces/<session>/drive.jsonl` in the
  [private state directory](#session-storage).
- A run drives one session.

## Documentation

[Design Decisions](DESIGN.md) |
[B Machines](docs/design/b-machines.md) |
[Runtime Specification](docs/spec/circuit-runtime.tex) |
[Harness Notes](docs/HARNESS.md) |
[Development](docs/DEVELOPMENT.md) |
[Testing](docs/TESTING.md) |
[Contributing](CONTRIBUTING.md) |
[Changelog](CHANGELOG.md) |
[PR/FAQ](prfaq.pdf)

## License

MIT
