# Roadmap

This file is the durable index of planned Circuit work. It summarizes roadmap
commitments and unresolved product work that were previously scattered across
the PR/FAQ, risk register, run contract, harness notes, and design notes.

The roadmap is ordered by dependency and product risk, not by promised dates.
Accepted architecture remains in
[`../architecture/decisions.md`](../architecture/decisions.md). Evidence and
open risks remain in [`risks.md`](risks.md). Product hypotheses and release
priorities remain in [`../../prfaq.tex`](../../prfaq.tex).

## 1. Validate comparative value

Run guided and unguided versions of realistic workflows and compare outcome
quality, supervision burden, recovery cost, and completion time. Continue
internal dogfood traces and customer interviews. Further investment depends on
evidence that Circuit intercepts meaningful mistakes or reduces supervision,
not merely that an agent can complete a machine.

Sources: `prfaq.tex` validation plan and metrics; `docs/project/risks.md`
guided-outcome risk.

## 2. Make check evidence trustworthy

Add workspace preflight so environment failures cannot masquerade as valid
workflow evidence. Implement the designed session-boundary provenance guard for
check sources and bindings. Define freshness, timeout, and diagnostic semantics
for checks before expanding them to CI state, review threads, and artifact
analysis.

Sources: `prfaq.tex` technical-risk discussion and Check bindings feature;
`docs/project/risks.md` external-observation and workspace-readiness risks.

## 3. Exercise richer workflow decisions

Test workflows with several valid transitions, ambiguous agent responses, and
choices that require gathering new evidence. Use those traces to harden event
extraction, blocked-event re-prompting, and state guidance.

Source: `docs/project/risks.md` branching and event-extraction risk.

## 4. Define intervention and recovery

Design an auditable operator path for exhausted retries and stuck non-terminal
sessions. The design must say how an operator can amend evidence or guidance,
escalate, resume, or terminate without silently bypassing machine authority.

Source: `docs/project/risks.md` recovery risk.

## 5. Harden concurrent sessions

Add safe concurrent-write behavior and recovery from interrupted session writes.
Improve session naming and selection, and associate Circuit sessions with
harness sessions without moving workflow semantics into an adapter.

Source: `docs/project/risks.md` concurrent-session risk.

## 6. Improve machine authoring and handoff

Measure raw Circuit-B authoring against agent-assisted authoring. Mature the
z-spec Circuit-B profile path, test end-to-end handoff into Circuit, and improve
templates and diagnostics when authoring effort is the limiting factor. Keep
human review and ProB validation mandatory.

Source: `prfaq.tex` authoring, validation, and technical-risk sections.

## 7. Make run configuration first-class

Add per-run prompt overrides or named prompt presets instead of requiring
worktree edits. Preserve the boundary among formal machine semantics, external
checks, state guidance, and the task supplied for a particular run.

Sources: `prfaq.tex` Per-run prompt overrides feature;
`docs/operations/run-contract.md` open items.

## 8. Stabilize traces and operator UX

Replace ad hoc driver trace maps with a stable typed event schema. Improve
status output and blocked-transition diagnostics, and add convenient harness UX
around driven workflows without weakening the CLI contract.

Sources: `docs/operations/run-contract.md` open items;
`docs/project/risks.md` recovery and session UX risks.

## 9. Expand harness integrations

Deepen the Claude Code integration, then opencode, using each harness's native
extension model. Distinguish tool-call guidance from drive-mode enforcement in
all user-facing claims. Do not duplicate transition logic outside the Go
runtime.

Sources: `prfaq.tex` delivery order; `docs/operations/harnesses.md`;
`docs/architecture/decisions.md` adapter constraints.

## 10. Resolve remaining formal-profile questions

Decide the preferred B operation style, terminal-state declaration and
non-terminal-stall property, justified Circuit-B type expansions, and whether B
refinements remain development-only. Each profile expansion requires an ADR and
corresponding parser, evaluator, and formal-gate coverage.

Sources: `docs/architecture/b-machines.md` open questions;
`docs/architecture/decisions.md` Circuit-B profile constraints.
