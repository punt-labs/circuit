# Contributing to Circuit

Thank you for improving Circuit. This file is the contributor entry point;
end-user setup and usage remain in [`README.md`](README.md).

## Development environment

Circuit is Nix-first. Enter the reproducible shell before building, testing, or
using project-planning tools:

```bash
nix develop
```

The shell supplies the Go and Node toolchains, linting tools, ProB, z-spec,
Beads, GitHub CLI, and supporting shell tools. Non-Nix development remains
possible when compatible tools are already installed. Exact requirements and CI
behavior are documented in [`docs/development.md`](docs/development.md).

## Repository documentation

Read these references before changing their respective surfaces:

- [`docs/architecture/decisions.md`](docs/architecture/decisions.md) — accepted
  architectural decisions
- [`docs/operations/run-contract.md`](docs/operations/run-contract.md) — driven
  agent contract
- [`docs/operations/harnesses.md`](docs/operations/harnesses.md) — harness
  integration behavior
- [`docs/testing.md`](docs/testing.md) — test layers and required gates
- [`docs/README.md`](docs/README.md) — full documentation index and naming
  conventions

Shared agent instructions live in [`AGENTS.md`](AGENTS.md). `CLAUDE.md` is only
a Claude Code entry point for those shared instructions.

## Development practices

- Keep changes small and easy to review.
- Write failing tests before implementation changes.
- Keep harness adapters thin; the Go CLI owns engine behavior.
- Prefer structural validation while state-machine design is unsettled.
- Do not add scheduling, GitHub API, MCP, or persistence behavior without a
  separate architectural decision.
- Use **Circuit** for the product and `circuit` for commands, paths, packages,
  and identifiers.
- Put detailed documentation under the appropriate lowercase `docs/` path.

## Local gates

Run the aggregate implementation gate before every commit:

```bash
make check
```

Run the structural Go quality gate for implementation changes:

```bash
make check-go-quality
```

Additional gates apply by surface:

```bash
make check-specs       # formal runtime specification changes
make check-machines    # B-machine changes
make smoke-pi          # pi extension behavior, when credentials are available
make smoke-drive       # Circuit-driven pi behavior, when credentials are available
```

See [`docs/testing.md`](docs/testing.md) for what each gate covers and its
external dependencies. `make help` is the authoritative catalog of Make
targets.

## Pull requests

Before opening a pull request:

1. Run all applicable gates.
2. Exercise changed user-facing behavior through its real entry point.
3. Update `CHANGELOG.md` under `Unreleased` for notable changes.
4. Update the README for user-facing behavior and an ADR for architectural
   decisions.
5. Keep the branch free of generated or temporary artifacts not intentionally
   tracked by the repository.

Pull requests should explain what changed, why it changed, and how the behavior
was verified.
