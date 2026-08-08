# Testing

`circuit` tests the state-machine engine and the harness adapters at different
levels. Keep the lower layers fast and automated; keep harness-level tests small
and explicit.

## Test pyramid

- Unit: pure validation and summary logic. Automated Go package tests.
- CLI smoke: user-facing commands work on sample files. Automated Go CLI tests.
- Example fixtures: checked-in playbooks remain valid. Automated fixture tests.
- Nix shell: project toolchain supplies Go, Beads, and lint tools. Manual smoke.
- Beads integration: repo uses central DoltDB with the circuit prefix. Manual
  smoke.
- pi extension smoke: project extension loads and validates a playbook. Manual
  smoke in pi and tmux.
- Cross-harness behavior: pi, Claude Code, and opencode load instructions as
  expected. Manual and documented.

## Unit tests

Unit tests live next to Go packages and should not need the network, GitHub,
Beads, pi, opencode, Claude Code, or shell commands.

They cover:

- YAML model parsing
- structural validation
- diagnostics
- summaries

These run as part of `make check`.

## CLI smoke tests

CLI smoke tests exercise the public command layer without requiring external
services. They verify that commands accept files, reject missing files, and emit
expected success or error status.

Current commands:

- `validate`
- `summary`

These run as part of `make check`.

## Example fixtures

Every checked-in example playbook should remain valid unless it is explicitly a
negative fixture. Positive examples are covered by automated tests that run the
same parse and validation path used by the CLI.

## Nix shell smoke

The Nix shell is the supported development environment. It should provide the
project toolchain, including Go and Beads.

This layer is currently checked manually when bootstrapping or changing the
flake. It does not need to run in every Go test because it validates the
development environment, not the engine.

## Beads integration smoke

Beads is external state backed by the Punt Labs Hosted DoltDB instance. It is not
a unit-test dependency.

Manual smoke checks should verify:

- `bd` comes from the Nix shell
- issue IDs use the `circ` prefix
- repo scoping uses the `repo:circ` label

Do not make normal tests depend on the production Beads database.

## pi extension smoke

The project-local pi extension is tested as a smoke test in a real pi session.
This is intentionally not part of `make check` yet because pi TUI testing is an
integration concern and depends on trust/session behavior.

A valid smoke run verifies:

- pi starts from the `circuit` repo
- project-local extension loading is approved
- the startup view lists the circuit extension
- the circuit validation command accepts an example playbook
- the command returns a successful validation message

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
