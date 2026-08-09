# Circuit Agent Instructions

This file is the harness-neutral instruction source for the `circuit` repo.

## Project purpose

`circuit` is a tiny formal state-machine engine for agent workflow loops. It
uses B-Method abstract machines as the formal model for workflow definitions.

## Mandatory reading

- `docs/DEVELOPMENT.md`
- `docs/TESTING.md`

## Development rules

- Do not quote blocks of code, config, command output, or error payloads in
  chat; summarize and reference files instead.
- Use the Nix dev shell for build, test, lint, and Beads work.
- Keep changes small and easy to review.
- Run `make check` before committing.
- Prefer structural validation over execution until the state-machine design is
  settled.
- Do not add scheduler, GitHub API, MCP, or persistence behavior without a
  separate design decision.

## Current useful commands

- List available machines with `circuit list`.
- Start an active circuit with `circuit start <machine>`.
- Report active circuit status with `circuit status`.
- Advance with `circuit advance <event>`.
- Run the full local gate with `make check`.
- Run the B-machine development gate with `make check-machines`.
