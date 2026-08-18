# Testing

`circuit` tests the B-machine engine, the runtime, the RPC protocol, the pi
extension, and the B machines themselves. Every tier is automated.

## Testing pyramid

```text
make check (automated, no external deps):
  Go unit tests        circuitb, circuitrun, circuitrpc
  Go CLI tests         cmd/circuit
  TS unit tests        .pi extension parsing/routing
  TS typecheck/lint    .pi extension types and style
  golangci-lint        Go lint
  markdownlint         docs

make check-specs (automated, requires z-spec + ProB):
  z-spec type-check + ProB model-check for formal design specs

make check-machines (automated, requires ProB):
  ProB init + model-check for each .mch

make smoke-pi (automated, requires pi + model API key):
  pi RPC smoke: list, start, status, advance against real pi

make coverage (automated, shows all tier summaries):
  Go engine coverage
  Go RPC protocol coverage
  TS pi extension coverage
```

## Go unit tests

Unit tests live next to Go packages and should not need the network,
GitHub, Beads, pi, opencode, Claude Code, or shell commands.

### `internal/circuitb`

Circuit-B parser, resolver, profile validator, and evaluator:

- B-machine lexing/parsing
- name/type resolution
- profile validation and rejection of unsupported constructs
- precondition evaluation with enums, NAT, BOOL
- state/advance result computation

### `internal/circuitrun`

Runtime with session lifecycle:

- session states: unloaded, active, suspended, stopped
- auto-stop on terminal state
- suspend/resume serialization
- start/status/advance/stop
- check binding execution and registry validation
- error paths for missing machines, malformed files, unknown entries

### `internal/circuitrpc`

RPC protocol logic extracted from the circuit-drives-pi spike:

- prompt formatting from status reports
- operation extraction from agent responses
- terminal state detection
- message text extraction from JSONL events
- fake-pi integration test: full runner loop against canned responses

## Go CLI tests

CLI tests exercise the public command layer: list, start, status,
advance, stop, help, and error paths for missing/extra arguments.

## TypeScript unit tests

Pi extension tests cover command parsing, routing, and context logic:

- `parseCircuitCommand` for all verbs, defaults, whitespace handling
- `parseCircuitStatus` for CLI output parsing
- `formatContextInjection` for agent context text
- `parseAdvanceOutput` for advance result parsing
- `formatToolResult` for tool result formatting

Run with vitest. Coverage reported via `@vitest/coverage-v8`.

## Formal design spec gate

The runtime design spec is a Z model of Circuit's totalized runtime operations.
Validate it with:

```bash
make check-specs
```

The target runs `z-spec check docs/spec/circuit-runtime.tex` and ProB
model-checking at the bounded mixed scope recorded in the spec. The Makefile
names each scope value so reviewers can see and override it:

```text
RUNTIME_SPEC_MACHINES=2
RUNTIME_SPEC_SESSIONS=2
RUNTIME_SPEC_CHECKS=2
RUNTIME_SPEC_MAX_INITIALISATIONS=15
```

The model-check target calls `probcli` directly because z-spec does not yet
expose ProB's `MAX_INITIALISATIONS` option, which this spec needs to certify the
intended initial configurations completely.

## B-machine gate

Every checked-in B machine passes ProB init and model-check:

```bash
make check-machines
```

Requires ProB (`probcli`). Not included in `make check` because ProB
is a development dependency, not a runtime dependency.

## Pi RPC smoke test

Automated pass/fail test against a live pi process:

```bash
make smoke-pi
```

Requires pi binary and a model API key. Tests the full extension
command surface: list, start, status, advance through pi RPC. Exits
0 on success, 1 on failure.

## Coverage targets

Non-command packages should maintain ≥85% statement coverage:

```text
internal/circuitb     ≥85%
internal/circuitrun   ≥85%
internal/circuitrpc   ≥85%
```

CLI and pi extension coverage is reported but not gated because those
packages are thin adapters over the core runtime.

Run `make coverage` to see all tier summaries.

## Continuous integration

GitHub Actions runs the project through the Nix dev shell on every push and pull
request:

```text
make check
make check-go-quality
make check-machines
```

The workflow uses the same pinned Go, Node, golangci-lint, and ProB toolchain as
local Nix development. ProB 1.15.1 is packaged by `nix/probcli.nix`; CI runs on
x86_64 Linux because upstream does not publish Linux aarch64 binaries.

## Required gates

Before committing code changes:

- `make check`
- `make check-go-quality`

Before changing formal design specs:

- `make check-specs`

Before changing B machines:

- `make check-machines`

Before changing pi extension behavior:

- `make smoke-pi` (when pi and credentials are available)
