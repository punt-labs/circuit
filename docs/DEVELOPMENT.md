# Development

`circuit` is Nix-first, with non-Nix development kept possible.

## Nix development

Use the project dev shell for normal work. It provides Go, Beads, GitHub CLI,
markdown linting, and other project tools.

The required local gate is `make check`.

## Non-Nix development

Non-Nix development should remain possible for contributors who already have the
required tools installed. The main requirements are Go 1.26 or newer,
golangci-lint, Node/npm for TypeScript and markdown lint, and GNU or BSD make.
Run `make tools` to install golangci-lint.

Nix is the reproducible development environment, not a runtime requirement for
the `circuit` binary.

## Beads

Project planning uses Beads with the `circ` issue prefix and `repo:circ` label.
Run Beads from the Nix dev shell so the CLI comes from the repo toolchain.

## Quality gate

Run the full gate before committing implementation changes. `make check`
auto-formats Go and TypeScript, then runs golangci-lint (govet + staticcheck +
unused + gofmt), Go tests with race detection and coverage, TypeScript
typecheck/lint/test, and markdown linting.

## Harness work

Harness behavior is documented in `docs/HARNESS.md`. Testing expectations are
documented in `docs/TESTING.md`. Keep harness adapters thin: the Go CLI owns the
engine behavior, and harness-specific code should wrap that CLI rather than
reimplement it.
