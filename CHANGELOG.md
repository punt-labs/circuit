# Changelog

All notable changes to circuit will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- Working Backwards PR/FAQ (`prfaq.tex`, `prfaq.bib`, compiled `prfaq.pdf`) at
  hypothesis stage v1.0, peer-review PASS, with cited evidence base under
  `research/` (gitignored)
- MIT `LICENSE`
- Working Backwards stage badge in `README.md`, linking to the compiled PDF
- `make docs-pdf` and `make clean-latex` targets for rebuilding the PR/FAQ
  PDF and sweeping LaTeX intermediates

### Changed

- `make check-docs` now excludes the generated `research/` directory from
  markdown linting
- `.gitignore` covers the generated `research/` cache
