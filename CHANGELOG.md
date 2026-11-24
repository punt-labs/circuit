# Changelog

All notable changes to circuit will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Fixed

- Prevent concurrent Circuit commands in one workspace from corrupting or
  losing session state (PL-1331).
- Bound the wait for the diagnostic rotation lock at startup. If the lock
  stays busy, Circuit warns once and runs without file diagnostics (PL-1342).
- Resolve the telemetry opt-out through Circuit's home-path resolver without
  changing the marker location or opt-out behavior (PL-1456).

### Security

- Clear 11 open Dependabot alerts (3 high). Go: `google.golang.org/grpc`
  1.80.0 → 1.83.2, `go.opentelemetry.io/otel` modules 1.43.0 → 1.47.0,
  `golang.org/x/net` 0.52.0 → 0.59.0. Pi extension (`.pi/package-lock.json`):
  eslint 10.8.1 → 10.15.0 and typescript-eslint 8.66.0 → 8.73.0, which move
  minimatch 10.2.6 → 11.0.9.

- Clear 7 open Dependabot alerts (2 high) in the pi extension
  (`.pi/package-lock.json`): vite 8.2.1 → 8.3.5, source-map-js 1.2.1 → 1.2.2,
  vitest and `@vitest/coverage-v8` 4.1.10 → 4.1.11.

### Added

- Private rotating Circuit CLI diagnostics and local, opt-out usage telemetry.
- Working Backwards PR/FAQ (`prfaq.tex`, `prfaq.bib`, compiled `prfaq.pdf`) at
  hypothesis stage, with the researcher's evidence base tracked under
  `research/`; revised to v2.0 within this unreleased cycle (see Changed)
- MIT `LICENSE`
- `circuit version [--json]` prints the build version
- `install.sh` installs a released binary to `~/.local/bin` after checking its
  SHA-256 against the release `checksums.txt`, and falls back to building the
  tagged release from source
- Release workflow: on a `v*` tag, runs the gates, publishes binaries for macOS
  and Linux (amd64 and arm64) with `checksums.txt`, then runs `install.sh`
  against the published release
- Working Backwards stage badge in `README.md`, linking to the compiled PDF
- `make docs-pdf` and `make clean-latex` targets for rebuilding the PR/FAQ
  PDF and sweeping LaTeX intermediates

### Changed

- **Breaking:** Circuit-managed state moved from `.tmp/sessions/`,
  `.tmp/circuit.suspended.json`, and `.tmp/circuit/<id>/drive.jsonl` to
  `sessions/`, `circuit.suspended.json`, and `traces/<id>/drive.jsonl`
  under `.punt-labs/local/circuit/` in Git workspaces, or
  `~/.punt-labs/circuit/data/workspaces/<key>/` outside Git. Old files are
  untouched; see [Session storage](README.md#session-storage) for manual
  copy instructions. Check evidence paths did not change (PL-1457).
- `.gitignore` ignores ethos machine-local (`.local.*`, `local/`) and runtime
  files (mission locks, sessions, missions), matching the self-hosted remote
- PR/FAQ revised to v2.1: z-spec named as the shipped authoring layer for
  agent-drafted machines (external FAQs state the launch path; internal
  technical risks carry today's alpha status and the z-spec adoption
  record); comparative and handoff authoring tests added to the validation
  plan; `feat:zspec-profile` (circuit-profile target in `b-create`)
  promoted to Must Do; design-time-plus-runtime suite framing added to
  the competitive FAQ
- PR/FAQ revised to v2.0 after an autonomous hive review meeting (7 hot
  spots, all resolved REVISE by 4-0 consensus): customer redefined as
  supervising gate-disciplined teams seeking permission to go unattended;
  competitive framing shifted to an open-standard race with authoring
  throughput as a tested risk; Circuit-B confirmed as the public primitive
  with agent-drafted machines as an endorsed layer; a designed
  session-boundary check-provenance guard replaces code-review-as-mitigation;
  revenue bet made accountable via a fifth internal-adoption metric;
  timeline resequenced around evidence with drive-vs-tool-call honesty and
  a no-MCP per-harness integration ruling; TAM terminal numbers dropped in
  favor of a measured interview funnel. Post-revision peer review fixes and
  a streamline pass included. Meeting record in
  `meetings/meeting-hive-summary-2026-08-30.md`
- `make check-docs` excludes the generated `research/` directory from
  markdown linting (agent-generated output, tracked but not lint-gated)
