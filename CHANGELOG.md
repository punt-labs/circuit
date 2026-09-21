# Changelog

All notable changes to circuit will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- `CONTRIBUTING.md` as the contributor entry point for development setup,
  practices, gates, and pull-request expectations
- A durable `docs/project/roadmap.md` that consolidates planned product,
  authoring, formal-method, and harness-integration work with links to its
  source documents
- Working Backwards PR/FAQ (`prfaq.tex`, `prfaq.bib`, compiled `prfaq.pdf`) at
  hypothesis stage, with the researcher's evidence base tracked under
  `research/`; revised to v2.0 within this unreleased cycle (see Changed)
- MIT `LICENSE`
- Working Backwards stage badge in `README.md`, linking to the compiled PDF
- `make docs-pdf` and `make clean-latex` targets for rebuilding the PR/FAQ
  PDF and sweeping LaTeX intermediates

### Changed

- The root README now focuses on end users; contributor and harness-development
  details link to their dedicated documentation
- Documentation is organized under lowercase `docs/` paths by architecture,
  operations, and project status, with standardized product-name capitalization
- Documentation now provides a direct quick start, documents the implemented
  `drive` path and gate boundaries, distinguishes historical design notes from
  current behavior, and updates the risk register to reflect completed
  multi-step workflow evidence without volatile test counts
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
