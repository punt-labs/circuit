# Reflections

This file records what the `tdd-flow` dogfood work taught us about Circuit,
Circuit-driven Pi, TDD slice delivery, observability, and quality gates.

## What worked

The B-machine layer held up. The machine stayed simple and formal while the
project-specific behavior lived outside B:

- `tdd-flow.mch` defines states, transitions, and abstract BOOL facts.
- `tdd-flow.checks.yaml` binds B facts to check names.
- `check-registry.yaml` maps check names to this repo's commands.
- `tdd-flow.prompts.yaml` maps B states to driver prompts.

The final useful TDD shape is:

```text
spec -> red -> green -> qualityReview -> done
                         qualityReview -> refactoring -> green
```

with two distinct gates:

- `testSuitePassed`
- `codeQualityPassed`

That distinction matters. Test green and code-quality green are not the same
fact.

## Important design changes discovered by dogfood

### `not(...)` belongs in Circuit-B

TDD red is naturally expressed as:

```text
not(testSuitePassed = TRUE)
```

Adding parenthesized `not(...)` simplified `tdd-flow` and removed the need for a
mandatory `failingTestObserved` B variable. Targeted red proof may still be
useful, but it is stricter evidence layered outside the machine, not required by
the formal workflow.

### Externally gated states are not terminal

A state with no currently enabled transitions is not necessarily terminal. If a
future check result can enable a transition, the session must remain active.
This mattered for `red`, where `implement` is blocked until tests pass.

### Refactoring must be connected to code quality

A soft prompt asking the agent to consider refactoring was not enough. The better
machine shape makes refactoring the path required when code quality fails:

```text
qualityReview -> done        requires codeQualityPassed = TRUE
qualityReview -> refactoring requires not(codeQualityPassed = TRUE)
```

This worked in practice. Once `make check-go-quality` failed, the driven Pi loop
entered repeated `refactoring -> green -> qualityReview` cycles until the quality
gate passed.

## Circuit-driven Pi worked

The useful architecture is:

```text
outer Pi or human -> Circuit -> inner Pi
```

Circuit owns the workflow. Pi performs work and requests transition events.
Circuit validates the requested transition and check results.

We now have:

- `PiRPCBackend`
- `RunGuidedSession`
- `circuit drive <machine> --task ...`
- machine-local prompt files such as `machines/tdd-flow.prompts.yaml`
- real `make smoke-drive` coverage for Circuit driving Pi through `build-job`

The driven TDD slices delivered real `--json` work.

## Prompt files should be data, not code

Hardcoding machine-specific guidance in the CLI was the wrong direction. The
right shape is a companion file:

```text
machines/<machine>.prompts.yaml
```

This mirrors check bindings:

```text
B boolean -> check binding
B state   -> prompt binding
```

Circuit stays generic. Machines, prompts, and checks are open for extension.

## Observability was essential

Before tracing, we could see only final diffs and final check state. That was not
enough to know whether Pi actually followed TDD.

`circuit drive` now writes:

```text
.tmp/circuit/<session-id>/drive.jsonl
```

The trace records:

- prompt text
- Pi response text
- workspace status after each Pi turn
- transition advances

This changed the discussion. It showed when Pi wrote only a test, when it wrote
production code, when it chose `finish`, and when it chose `refactor`.

## Worktree setup matters

One TDD smoke failed for a misleading reason: the fresh worktree did not have
`.pi/node_modules`, so `make check` failed before Pi did anything useful. That
made `not(testSuitePassed = TRUE)` true for the wrong reason.

Lesson: Circuit-driven smoke tests need a prepared worktree. Otherwise check
failures may reflect environment setup, not agent work.

## What the `--json` dogfood delivered

The current command surface now has JSON output for:

- `list --json`
- `load --json`
- `scaffold --json`
- `start --json`
- `status --json`
- `advance --json`
- `stop --json`
- `unload --json`

The later slices were delivered by real Circuit-driven Pi runs and then applied
back to the main branch after `make check` passed.

## Code quality gate lessons

A `codeQualityPassed` fact is useful only if it has real teeth.

The first structural quality gate was calibrated for testing, not final policy:

```text
dupl threshold: 100
gocognit min-complexity: 20
funlen statements: 40
funlen lines: 70
```

This produced a small failing set:

```text
dupl: 2
funlen: 2
gocognit: 2
```

After the driven refactor loop, `make check-go-quality` passed.

This confirms the mechanism: code-quality failure can force refactoring loops.
The exact thresholds and tools can be tightened later.

## Current remaining questions

- How strict should TDD red proof be beyond `not(testSuitePassed = TRUE)`?
- Should `drive.jsonl` capture check command stdout/stderr, not only boolean
  check results?
- Should prompt overrides be first-class input for a single run rather than
  editing `machines/tdd-flow.prompts.yaml` in a worktree?
- Should `/circuit:tdd <slice>` in outer Pi wrap `circuit drive tdd-flow`?
- Should `check-go-quality` include more tools after dupl/gocognit/funlen?

## Quality-ratchet loop lessons

The ratchet loop idea works. The approach:

1. Create a worktree from current HEAD.
2. Prepare the worktree (`make build`, `npm install`).
3. Run `circuit drive tdd-flow` with a ratchet-specific prompt override.
4. Apply and verify the result.
5. Commit.
6. Repeat with the next quality dimension.

Three ratchet loops ran successfully:

- goconst (run 1): worked first time using generic TDD prompt.
- gocritic (run 2): generic prompt failed; ratchet prompt override succeeded.
- maintidx (run 3): ratchet prompt override succeeded.

`make check-go-quality` now runs:

```text
dupl, gocognit, funlen, goconst, gocritic, maintidx
```

and passes.

## Generic TDD prompt is wrong for ratchet slices

The standard `spec` prompt says:

```text
Write the failing test.
Do not implement production code.
When the targeted test is failing, request writeTest.
```

For a ratchet slice, there is no failing product test. The work is:

```text
Tighten the quality gate so check-go-quality fails on current code.
Then refactor until it passes.
```

The agent kept requesting `writeTest` without changing files, hitting the
10-prompt cap with nothing to show.

The fix was a prompt override in the worktree. However, editing prompt files in
a worktree is a manual step that breaks the loop ergonomics.

## Needed: first-class prompt override per run

The goal for tomorrow is to make prompt override input rather than file editing.

Something like:

```text
circuit drive tdd-flow \
  --task "Add gocritic to check-go-quality" \
  --prompt-spec ratchet
```

where `ratchet` is either a named preset or a file path:

```text
circuit drive tdd-flow \
  --task "Add gocritic to check-go-quality" \
  --prompts .tmp/ratchet-prompts.yaml
```

This makes prompt overrides data, not worktree edits. It also closes the loop
without requiring manual worktree prompt file changes between runs.

## Ratchet prompt override structure

The ratchet prompt override rewrites the standard states:

```yaml
states:
  spec:
    prompt: >
      Tighten check-go-quality so it fails on current code. Do not refactor yet.
      When check-go-quality fails, request writeTest.
    event: writeTest
  red:
    prompt: >
      Refactor until make check and check-go-quality pass. Request implement.
    event: implement
```

`green`, `qualityReview`, and `refactoring` can remain the same as the product
TDD prompts.

## Worktree setup as a prerequisite

Every smoke/ratchet run must include:

```text
git worktree add .tmp/worktrees/<name> HEAD
env -C .tmp/worktrees/<name>/.pi npm install
env -C .tmp/worktrees/<name> make build
```

Without `npm install`, `make check` fails for setup reasons, which makes
`testSuitePassed` false for the wrong reason and breaks the TDD semantics.

This three-step setup should become `.bin/prep-drive-worktree.sh`.

## Current conclusion

Circuit is useful when it is the middle authority, not merely context injected
into the agent. The strongest pattern so far is:

```text
human chooses slice
Circuit drives Pi through tdd-flow
checks decide progress
quality gate forces refactoring
trace proves what happened
```
