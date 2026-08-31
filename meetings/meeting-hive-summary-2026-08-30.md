# PR/FAQ Review Meeting Summary

**Date:** 2026-08-30
**Document:** prfaq.tex
**Scope:** Full meeting (7 hot spots)
**Mode:** Hive (autonomous consensus, Agent Teams)

## Overall Assessment

**Opening (Alex):** The 38.33% constraint violation and 36.49% instruction-following failure numbers are externally sourced, so unattended agent workflows have a measurable discipline gap; that proves the general problem, and it says nothing about this segment's size while trust in AI output sits at a record low 29%. The Risk Assessment says it plainly: Value is HIGH on problem while the moat is focus. An MIT license with no structural moat, shipped publicly, is a gift wrapped for the fastest well-funded incumbent to absorb. The built part is the low-risk part, the Claude Code adapter is Should Do and first to cut, and the only customer today is Punt Labs itself. What changes this read is the ten interviews and a month of dogfood traces landing anything other than empty, with kill criteria enforced when they land.

**Closing (Alex):** Two of my three questions shifted, one did not earn it yet. Problem-worth-solving stayed unresolved on the segment question: redefining the customer from "runs agents unattended" to "gate-disciplined teams who want a reason to stop supervising" is the right change, and it stays a renamed hypothesis until the interview funnel screens both segments and produces a number comparable against the 29% trust figure. Strong-differentiated-solution shifted the most: reframing copyability as category validation is candid instead of hopeful, and a same-session check-provenance guard in place of "code review will catch it" closes the one gap that would have let tdd-flow ship a self-grading agent. Build-it-now shifted less than the decision table suggests: item 6 concedes the engine already runs at low risk, so "harden first" was never really a gate; there is still no answer for why the Claude Code adapter stays Should Do, and nothing yet says what happens if Punt Labs itself misses the internal-adoption checkpoint. Reconvene after the ten interviews and the dogfood month land, with the funnel-screening numbers for both segments, the first read on the internal-adoption checkpoint, and the operator's ruling on B-versus-surface and MCP-as-universal-adapter in hand.

## Decisions

| # | Hot Spot | Door | Decision | Resolution | Winning Argument | Dissent |
|---|----------|------|----------|------------|------------------|---------|
| 1 | Focus-only moat lacks capacity and sequencing backing | one-way | REVISE | CONSENSUS | Wei: the velocity claim rests on a few-hours-a-week authoring capacity disclosed two FAQs later; Dana: copyability validates the category for an open standard | None (4-0) |
| 2 | Customer defined by an unattended habit the trust data contradicts | two-way | REVISE | CONSENSUS | Dana and Priya: the buyer is the supervising team seeking permission to stop watching, and 68% daily use at 29% trust describes watchers | None (4-0) |
| 3 | B-Method public authoring surface is an unmade decision | one-way | REVISE | CONSENSUS | Alex: the versioned customer artifact, B itself or a surface compiling to B, carries different failure recoveries and the text picked one without noticing | None (4-0) |
| 4 | Check-independence mitigation guards bindings, never content | one-way | REVISE | CONSENSUS | Dana: the sharpest risk is defended by code review, the soft mechanism the Problem section indicts; one machine-checked session-boundary guard replaces it | None (4-0) |
| 5 | Revenue "None, deliberately" asserted without accountability | two-way | REVISE | CONSENSUS | Alex: an asset cannot anchor faq:revenue and cut first in faq:timeline; add an internal-adoption metric with checkpoint and consequence | None (4-0) |
| 6 | Delivery order gates learning behind a finished task | two-way | REVISE | CONSENSUS | Dana: "harden first" is a checked box dressed as a gate; Priya: tool-call mode is compliance-dependent and drive mode is pi-only, and the reader learns too late | None (4-0) |
| 7 | TAM base uncited while terminal numbers anchor | two-way | REVISE | CONSENSUS | Alex and Wei: hedge language is never a citation; cite the 30M base or drop the 20-30K range for the funnel measurement | None (4-0) |

## Revision Queue (for /prfaq:feedback)

### Directive 1: Reframe the moat as an open-standard race and back it with capacity

In faq:competitive and the spokesperson quote, state the distribution-velocity strategy with conviction: being copied by Sponsio validates the category while circuit is already running in production. Name the prize: the machine format, trace format, and check-registry pattern becoming the convention every harness assumes. Connect the "shipping working machines for the loops teams run every day" claim to the authoring capacity disclosed in faq:scaling and faq:cost-to-run. Name authoring throughput as a tested risk. State the MIT license as irreversible. Define whether the public GA precedes or follows the interview month.

### Directive 2: Redefine the customer as the supervising team seeking permission

In the lede, faq:what, and the customer quote, replace "run AI coding agents unattended" as a present-tense habit with the actual buyer: teams with CI gate discipline that run agents daily, still supervise every run, and want a credible reason to stop. Dramatize the watching-becomes-trusting transition in the Maya quote. Insert the missing funnel step in faq:tam (fraction of gate-disciplined teams already unattended, reported as unknown). Redesign faq:next-step to screen both segments (already-unattended vs. blocked-by-trust) and count the recruiting funnel itself as the segment-size measurement. Add unattended-operation safety answers to faq:technical-risks: cost bounds, stall detection, blast-radius limits.

### Directive 3: Decide the authoring surface and take a position on agent-authored machines

Operator ruling (2026-08-30): both. Raw Circuit-B stays the public primitive and the versioned customer artifact, and friendlier use-case layers (agent-drafted machines, compiled front-ends) build on top of it rather than replacing it. Revise faq:need-b, faq:why-b-method, and feat:no-full-b to state that layering plainly. Take an explicit position on agent-drafted machines (the likeliest path to machine six), including whether ProB verification plus human diagram review is the intended authoring loop. Make the faq:next-step authoring exercise comparative and agent-assisted rather than unassisted raw B. Connect the ARM verification gap to its cost: the edit loop for machine authors on Apple Silicon.

### Directive 4: Close the same-session check-provenance gap and narrow the headline

In faq:technical-risks and feat:checks, replace code-review-as-mitigation with a machine-checked guard: circuit refuses a check whose source file has commits since the session started, unless pinned to an operator-reviewed hash. Distinguish binding provenance from check-content provenance. State plainly that tdd-flow's agent-written tests fall under the guard's session boundary, with content quality still bounded by the cited 26 to 68.1% generated-test failure rates. Add check-file authorship and last-modified fields to feat:trace-detail. Narrow the sub-headline so "tests pass" claims what circuit guarantees: the gate ran, on a command that predates the session, with the exit status recorded.

### Directive 5: Make the revenue bet accountable

Keep the free MIT engine. Rewrite faq:revenue around the format-gravity reason: give the format away until it is the default; whatever gets built on a default format is worth charging for; before that there is nothing to sell. Add the internal-adoption metric to faq:metrics: an ethos or beadle production pipeline running on circuit machines by a stated checkpoint, with a stated consequence for a miss. Set Viability to MEDIUM until that metric reports. Resolve the contradiction between "the enforcement primitive the rest of the Punt Labs toolchain stands on" and the beadle embedding cutting first. Add the customer-facing continuity answer to the external FAQs: machines and bindings are text files the team owns, and an abandoned MIT binary with no server still gates correctly.

### Directive 6: Resequence the timeline around evidence, and tell the reader which door they get

Replace "harden the engine" as phase one, since the risk table already states that scope runs today at low risk. Run CLI-path validation with interviewed teams in parallel with any hardening. In Getting Started, the CTA, and faq:what, distinguish the two integration modes where the reader stands: drive mode (the machine owns the loop, the unattended guarantee, pi-only today) versus tool-call mode (compliance-dependent until an adapter lands). Either promote the Claude Code adapter to Must Do or stop listing Claude Code as an unattended harness at launch. Defend the never-cut list as a premise (enforcement failure is existential, demand failure is recoverable) or change it. Operator ruling (2026-08-30) on the MCP question: no MCP surface. pi lacks native MCP support, and opencode offers a better integration path than Claude Code for certain capabilities, so deep per-harness integration is the deliberate strategy. State the two integration relationships the repo already defines, circuit drives the coding harness, and the coding harness (or an embedding product such as ethos) hosts circuit, and present per-harness adapters as a chosen investment with that rationale.

### Directive 7: Source the TAM base or drop the terminal numbers

Cite a dated developer-population source (SlashData-class) for the 30 million base, or drop the 20,000 to 30,000 terminal range and report segment size as unmeasured until the interview funnel runs. Soften or source the one-in-ten gate-discipline multiplier, which reads high against practitioner experience. Add one line stating format gravity as the reason any floor number is a floor. Keep Alex's retroactive test: if the funnel measurement lands more than 3x from the estimate, the top-down chain gets retired.

## Deferred Items

None. All seven hot spots reached consensus with zero escalations.

## Research Completed

None during the meeting. The document's existing evidence base (research/research-2026-08-30-circuit-prfaq.md) supplied all figures debated.

## Notes

Every hot spot drew four ITERATE verdicts in Round 1, the consensus position that maps to a REVISE decision in the table above; no rebuttal rounds were required and no one-way door produced a REJECT. Three cross-hot-spot patterns emerged. First, internal FAQs repeatedly contain the candid answer to a question the external FAQs raise, so the candor lands on the wrong audience: bus-factor (hot spots 1 and 5), integration reality (hot spot 6), and continuity-on-abandonment (hot spot 5). Second, several claims anchor one FAQ and are disclaimed in another without cross-reference: velocity vs. authoring capacity (hot spot 1), strategic value vs. cut-first (hot spot 5), hardening vs. already-low-risk (hot spot 6). Third, the document's strongest evidence often argues against its own framing one section away: the 29% trust figure both sells the timing and disproves the segment (hot spot 2), and the 68.1% generated-test figure both justifies the product and undermines its sub-headline (hot spot 4). Two decisions in the queue needed the operator, and both were ruled the same day. B-versus-surface: both, the primitive and its use cases; raw Circuit-B stays the public artifact with agent-drafted and compiled layers built on top. MCP-as-universal-adapter: rejected; pi lacks native MCP and opencode offers a better integration path than Claude Code for certain capabilities, so deep per-harness integration is the deliberate investment. The operator also expects circuit to drastically simplify ethos, which strengthens Directive 5's internal-adoption premise: ethos mission pipelines are the first embedding use case, and the internal-adoption metric measures exactly that bet.
