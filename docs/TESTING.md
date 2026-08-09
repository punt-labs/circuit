# Testing

`circuit` tests the B-machine engine, the runtime, and the harness adapters at
different levels. Keep the lower layers fast and automated; keep harness-level
tests small and explicit.

## Test pyramid

- Unit: Circuit-B parser, resolver, profile validation, and evaluator.
  Automated Go package tests.
- Runtime: suspend/resume, check bindings, advance/block behavior. Automated Go
  package tests.
- CLI: user-facing commands work on sample machines. Automated Go CLI tests.
- B-machine gate: checked-in machines pass ProB init and model-check. Requires
  ProB; run with `make check-machines`.
- pi extension smoke: project extension loads and runs circuit commands. Manual
  smoke in pi and tmux.
- Cross-harness behavior: pi, Claude Code, and opencode load instructions as
  expected. Manual and documented.

## Unit tests

Unit tests live next to Go packages and should not need the network, GitHub,
Beads, pi, opencode, Claude Code, or shell commands.

They cover:

- B-machine lexing/parsing
- name/type resolution
- Circuit-B profile validation
- precondition evaluation
- state/advance result computation
- BOOL preconditions and check binding behavior

These run as part of `make check`.

## Runtime tests

Runtime tests exercise the `circuitrun` package:

- suspend/resume lifecycle
- start/status/advance/stop
- check binding execution
- check registry validation
- error paths for missing machines, malformed suspended files, and unknown
  registry entries

These run as part of `make check`.

## CLI tests

CLI tests exercise the public command layer without requiring external
services. They verify that commands accept machines, reject missing machines,
and emit expected success or error output.

Current commands:

- `list`
- `start`
- `status`
- `advance`
- `stop`

These run as part of `make check`.

## B-machine gate

Every checked-in B machine should pass ProB init and model-check. Run with:

```bash
make check-machines
```

This requires ProB (`probcli`). It is not included in `make check` because
ProB is a development dependency, not a runtime dependency.

## pi extension smoke

The project-local pi extension is tested as a smoke test in a real pi session.
This is intentionally not part of `make check` yet because pi TUI testing is an
integration concern and depends on trust/session behavior.

A valid smoke run verifies:

- pi starts from the `circuit` repo
- project-local extension loading is approved
- the startup view lists the circuit extension
- `/circuit list` shows available machines
- `/circuit start build-job` starts an active circuit and reports state
- `/circuit advance start` advances the active circuit
- `/circuit status` reports the updated state

Use tmux for visible pi sessions so the pane can be inspected and stopped.

## Cross-harness behavior

Instruction loading behavior is documented in `docs/HARNESS.md`. Keep these
findings empirical: record what each harness actually loaded or executed, not
what we assume from another harness.

## Required gate

Before committing code changes, run the project gate through the Nix shell:

- `make check`

Before changing Nix, Beads, or harness behavior, also perform the relevant smoke
checks and record the result in the PR summary or docs.
