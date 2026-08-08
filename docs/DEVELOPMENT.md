# Development

`circuit` is Nix-first, with non-Nix development kept possible.

## Nix development

Use the project dev shell for normal work. It provides Go, Beads, GitHub CLI,
markdown linting, and other project tools.

The required local gate is `make check`.

## Non-Nix development

Non-Nix development should remain possible for contributors who already have the
required tools installed. The main requirements are Go 1.26 or newer,
staticcheck, Node/npm for markdown lint fallback behavior, and GNU or BSD make.

Nix is the reproducible development environment, not a runtime requirement for
the `circuit` binary.

## Beads

Project planning uses Beads with the `circ` issue prefix and `repo:circ` label.
Run Beads from the Nix dev shell so the CLI comes from the repo toolchain.

## Quality gate

Run the full gate before committing implementation changes. The gate covers Go
formatting, Go vet, staticcheck, markdown linting, and race-enabled Go tests.

## Harness work

Harness behavior is documented in `docs/HARNESS.md`. Testing expectations are
documented in `docs/TESTING.md`. Keep harness adapters thin: the Go CLI owns the
engine behavior, and harness-specific code should wrap that CLI rather than
reimplement it.
