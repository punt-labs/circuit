# Development

`circuit` is Nix-first, with non-Nix development kept possible.

## Nix development

Use the project dev shell for normal work. It provides Go, Node, Beads, GitHub
CLI, markdown linting, staticcheck (`go-tools`), shellcheck, and ProB 1.15.1 via
the custom `nix/probcli.nix` derivation. ProB is included only on systems where
ProB publishes a build (macOS and x86_64 Linux). golangci-lint is not in the dev
shell; install the pinned version with `make tools`.

The required local gates are:

```bash
make check
make check-go-quality
make check-machines
```

## Non-Nix development

Non-Nix development should remain possible for contributors who already have the
required tools installed. The main requirements are Go 1.26 or newer,
golangci-lint, Node/npm for TypeScript and markdown lint, and GNU or BSD make.
Run `make tools` to install golangci-lint.

Nix is the reproducible development environment, not a runtime requirement for
the `circuit` binary. ProB is a development/release dependency only.

## Beads

Project planning uses Beads with the `circ` issue prefix and `repo:circ` label.
Run Beads from the Nix dev shell so the CLI comes from the repo toolchain.

## Continuous integration

`.github/workflows/check.yml` runs pushes to `main` and `feat/**`, plus pull
requests targeting `main`, through the Nix dev shell. CI executes `make check`,
`make check-go-quality`, and
`make check-machines`. ProB is pinned to 1.15.1 and fetched from the official
HHU release host. CI runs on x86_64 Linux because ProB does not publish a Linux
aarch64 build.

## Releases

`.github/workflows/release.yml` runs when a `v*` tag is pushed. It requires the
tagged commit to be on `main` and `VERSION` in `install.sh` to match the tag.
It then runs the same gates as CI, builds the four binaries with `make dist`,
and stages them with `checksums.txt` as a workflow artifact. The installer
downloads and verifies the staged binary in an empty `HOME` on Linux and macOS
(amd64 and arm64). Only after all four checks pass does the workflow publish
those same assets to a GitHub release.

To cut a release:

1. Set `VERSION` in `install.sh` to the new version and update `CHANGELOG.md`.
2. Merge that change to `main`.
3. Tag the merge commit `v<version>` and push the tag.
4. Point the README Quick Start installer URL at a commit that contains the
   released `install.sh`.

The installer downloads from GitHub release assets and raw GitHub URLs, so it
works only while the repository is public.

## Quality gate

Run the full gate before committing implementation changes. `make check`
auto-formats Go and TypeScript, then runs golangci-lint (govet + staticcheck +
unused + gofmt), Go tests with race detection and coverage, TypeScript
typecheck/lint/test, and markdown linting.

## Make targets

The root Makefile is organized around product surfaces, not implementation
languages. `make help` lists every target.

- `check` runs the automated aggregate gate (format, engine, RPC, pi
  extension, docs).
- `check-engine` validates the Go engine and CLI.
- `check-rpc` validates the RPC protocol logic.
- `check-pi-extension` validates the project-local pi extension.
- `check-docs` validates Markdown documentation.
- `check-go-quality` runs the structural Go quality gate used by `tdd-flow`.
- `check-go-all` prints a wider Go linter report; it is informational, not a
  gate.
- `check-machines` validates B machines with ProB.
- `check-specs` validates the formal Z design spec with z-spec and ProB.
- `check-runtime-spec` type-checks the Circuit runtime Z spec with z-spec.
- `model-check-runtime-spec` model-checks the Circuit runtime Z spec with ProB.

Additional targets:

- `lint` — alias for `lint-engine`
- `test` — runs all tests: Go engine, RPC protocol, and pi extension
- `build` — alias for `build-engine`; writes `./circuit`
- `install` — builds and copies the binary to `~/.local/bin`
- `dist` — cross-compiles release binaries for macOS and Linux (amd64 and
  arm64) into `dist/`
- `lint-install` — runs shellcheck on `install.sh`
- `test-install` — runs `install.sh` against the local build in a scratch
  `HOME` (part of `check`)
- `docs` — alias for `check-docs`
- `docs-pdf` — rebuilds `prfaq.pdf` from `prfaq.tex` (needs `latexmk`)
- `clean-latex` — removes LaTeX build intermediates
- `format` — auto-formats Go and TypeScript
- `coverage` — shows coverage summary for all tiers
- `smoke-pi` — pi RPC smoke test (requires pi and a model API key)
- `smoke-drive` — `circuit drive` smoke test (requires pi and a model API key)
- `tools` — installs development tools (golangci-lint)
- `clean` — removes build artifacts

## Harness work

Harness behavior is documented in `docs/HARNESS.md`. Testing expectations are
documented in `docs/TESTING.md`. Keep harness adapters thin: the Go CLI owns the
engine behavior, and harness-specific code should wrap that CLI rather than
reimplement it.
