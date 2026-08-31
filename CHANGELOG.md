# Changelog

All notable changes to circuit will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- Working Backwards PR/FAQ (`prfaq.tex`, `prfaq.bib`, compiled `prfaq.pdf`) at
  hypothesis stage, with the researcher's evidence base tracked under
  `research/`; revised to v2.0 within this unreleased cycle (see Changed)
- MIT `LICENSE`
- Working Backwards stage badge in `README.md`, linking to the compiled PDF
- `make docs-pdf` and `make clean-latex` targets for rebuilding the PR/FAQ
  PDF and sweeping LaTeX intermediates

### Changed

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
