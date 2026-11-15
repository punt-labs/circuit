# Contributing

`circuit` is developed in the Nix dev shell. See
[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for non-Nix setup, CI, and the full
list of Make targets, and [docs/TESTING.md](docs/TESTING.md) for the testing
pyramid.

## Setup

Enter the dev shell:

```bash
nix develop
```

Install the pinned golangci-lint:

```bash
make tools
```

Install the pi extension's development dependencies:

```bash
npm --prefix .pi ci --no-audit --no-fund
```

## Quality Gates

Run these before every commit. CI runs the same three targets.

`make check` formats Go and TypeScript in place, then runs golangci-lint, the
Go and RPC tests, the pi extension typecheck, lint and tests, and markdownlint:

```bash
make check
```

Structural Go quality gate:

```bash
make check-go-quality
```

ProB init and model check of every machine in `machines/` (requires `probcli`):

```bash
make check-machines
```

## Optional Checks

Formal Z design spec (requires z-spec and ProB):

```bash
make check-specs
```

Live pi smoke tests (require pi and a model API key):

```bash
make smoke-pi
```

```bash
make smoke-drive
```

## Reviews

PR review follows
[punt-kit/standards/pr-review.md](https://github.com/punt-labs/punt-kit/blob/main/standards/pr-review.md).
