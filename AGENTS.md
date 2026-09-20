# Circuit Agent Instructions

This file is the harness-neutral instruction source for the `circuit` repo.

## Project purpose

`circuit` is a tiny formal state-machine engine for agent workflow loops. It
uses B-Method abstract machines as the formal model for workflow definitions.

## Mandatory reading

- `docs/development.md`
- `docs/testing.md`

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

## Circuit-managed workflows

- Use Circuit to manage any workflow for which a checked-in machine exists; do
  not merely perform the equivalent steps by hand.
- For pull requests, start `pr-watch` after opening the PR and keep its session
  active until the PR merges or is explicitly stopped.
- Before each workflow action, inspect the active session with the harness
  adapter's status operation or `circuit status <session>`, then follow the
  machine's current state and enabled operations.
- Request every state change with the harness adapter's advance operation or
  `circuit advance <event> <session>`. Never claim workflow progress unless the
  request succeeds.
- Treat blocked transitions as authoritative. Gather the required evidence or
  complete the required work, then retry; do not bypass or narrate past a block.
- Continue monitoring CI, reviews, review threads, fixes, and merge readiness
  through the same Circuit session. Opening a PR is not completion.
- When checks or review findings require work, advance to the machine's fixing
  path before editing. After fixes are pushed and evidence is current, request
  the next enabled transition.
- Stop or unload the session only after the workflow reaches its intended end or
  the operator explicitly abandons it.

## Current useful commands

- List available machines with `circuit list`.
- Start an active circuit with `circuit start <machine>`.
- Report active circuit status with `circuit status`.
- Advance with `circuit advance <event>`.
- Run the full local gate with `make check`.
- Run the B-machine development gate with `make check-machines`.
