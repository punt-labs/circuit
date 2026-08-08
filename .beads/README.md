# Beads (circuit)

Issue tracking uses `bd` against the Punt Labs central Hosted DoltDB instance.
There is no per-repo issue store. Issues are separated by the `circ` prefix and
`repo:circ` label.

## Source of truth

- Issues live in central Hosted DoltDB.
- `.beads/metadata.json` holds this repo's mode, database, and issue prefix.
- `.beads/config.yaml` configures directory auto-scoping and disables derived
  exports.
- Runtime/export files under `.beads/` are ignored and are not authoritative.

## Connection

Connection details come from the repo environment and user secret store. Secrets
must never be committed and must never enter the Nix store.

## Daily commands

Run commands from the Nix dev shell so `bd` comes from the repo toolchain.

Common operations:

- list this repo's beads
- view ready work
- create new circuit beads
- close completed beads

Expected issue IDs begin with `circ-`.
