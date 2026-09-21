# Documentation

Detailed Circuit documentation uses lowercase file names and is grouped by
purpose. `README.md` remains capitalized as the conventional index-file
exception at the repository root and within `docs/`.
Root-level user, contributor, and governance documentation is limited to the
conventional entry points `README.md`, `CONTRIBUTING.md`, `CHANGELOG.md`,
`AGENTS.md`, and `CLAUDE.md`. Detailed documentation belongs under `docs/`.

## Start here

- [`../README.md`](../README.md) — product overview and quick start
- [`../CONTRIBUTING.md`](../CONTRIBUTING.md) — contributor setup and practices
- [`development.md`](development.md) — development environment and local gates
- [`testing.md`](testing.md) — test layers, formal checks, and smoke tests

## Architecture

- [`architecture/decisions.md`](architecture/decisions.md) — accepted ADRs
- [`architecture/b-machines.md`](architecture/b-machines.md) — historical
  B-machine design rationale
- [`architecture/design-patterns.md`](architecture/design-patterns.md) — design
  principles used in the codebase
- [`spec/circuit-runtime.tex`](spec/circuit-runtime.tex) — formal runtime design
  specification

## Operations and integrations

- [`operations/run-contract.md`](operations/run-contract.md) — machine prompts,
  external checks, driver responses, evidence, and traces
- [`operations/harnesses.md`](operations/harnesses.md) — pi, Claude Code, and
  opencode integration notes

## Project status

- [`project/roadmap.md`](project/roadmap.md) — ordered planned work and source
  references
- [`project/risks.md`](project/risks.md) — evidence and remaining risks
- [`project/reflections.md`](project/reflections.md) — chronological dogfood and
  implementation lessons

## Naming convention

Use sentence case for headings. Capitalize the product as **Circuit** in prose
and headings. Use lowercase `circuit` only for the executable, package/module
names, command examples, literal paths, and identifiers. Use each external
product's official spelling: **pi**, **Claude Code**, and **opencode**.
