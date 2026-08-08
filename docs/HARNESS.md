# Harness Notes

`circuit` is Punt Labs' smallest cross-harness project. The product is a small
state-machine playbook validator; the repo also lets us compare how pi, Claude
Code, and opencode consume project instructions.

## Instruction files

The intended shared instruction source is:

```text
AGENTS.md
```

Harness-specific entrypoints may reference or layer on top of it.

## Claude Code

Claude Code uses:

```text
CLAUDE.md
```

For `circuit`, `CLAUDE.md` is intentionally a small stub that imports
`AGENTS.md` and reserves space for future Claude Code-specific notes.

We are not adding Claude Code hooks, subagents, or custom commands in the
initial scaffold.

## pi

Pi documentation says pi loads context files from global, parent, and current
directories, including:

```text
AGENTS.md
CLAUDE.md
AGENTS.override.md
```

Pi project resources under `.pi/` require project trust. Context files are loaded
unless context discovery is disabled.

For `circuit`, pi support should stay minimal at first:

- use the Nix dev shell
- use `AGENTS.md` / `CLAUDE.md` context behavior as observed
- use tmux for visible long-running terminal monitoring
- add a project-local pi extension only after the CLI behavior is useful enough
  to wrap

## opencode

Opencode supports `AGENTS.md` and can also use `opencode.json` for project
configuration. Do not add `opencode.json` until we actively test opencode in this
repo.

## Observed pi behavior

A pi startup check from the `circuit` repo showed these loaded context files:

```text
~/Coding/punt-labs/CLAUDE.md
~/Coding/punt-labs/circuit/AGENTS.md
```

With both `AGENTS.md` and `CLAUDE.md` present in `circuit`, pi loaded
`AGENTS.md` for the repo-local context and did not list the repo-local
`CLAUDE.md`. This supports the intended split: shared repo instructions in
`AGENTS.md`, Claude Code entrypoint/addendum in `CLAUDE.md`.

The same startup check warned that tmux is using `extended-keys-format xterm`;
pi recommends `csi-u` for best modified-key handling.

## Open questions

- Is the Claude Code `@AGENTS.md` stub enough, or do we need a different split?
- Does opencode need explicit `opencode.json` instructions, or is `AGENTS.md`
  sufficient?
- Should `AGENTS.override.md` be used for pi if shared instructions become too
  Claude-specific?

## Current policy

Do not centralize or render harness configuration yet. First, observe each
harness' native behavior in this small repo.
