# Circuit Agent Instructions

This file is the harness-neutral instruction source for the `circuit` repo.

## Project purpose

`circuit` is a tiny state-machine validator for agent workflow playbooks. The
current implementation is a prototype validator and summarizer, not a ratified
runtime engine design.

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

- Validate an example playbook with the CLI.
- Summarize an example playbook with the CLI.
- Run the full local gate with `make check`.
