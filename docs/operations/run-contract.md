# Run contract

This note describes what Circuit needs beyond the B machine when Circuit is
driving an agent backend. It reflects what the `tdd-flow` dogfood work actually
produced, not just the earlier sketch.

## The three companion files

Each machine can ship up to three data files. All are optional except when the
machine references them.

```text
machines/<name>.mch            formal workflow (states, transitions, guards)
machines/<name>.checks.yaml    binds B BOOL variables to check names
machines/<name>.prompts.yaml   binds B states to driver prompts
```

The B machine stays universal. Project-specific behavior lives in the check
registry and prompt file.

## Analogy

```text
B BOOL variable -> check binding
B state         -> prompt binding
```

Checks answer "what fact must be true for this transition to fire?"
State prompts answer "what should the driven agent do while we are here?"

## Minimal shape used in Circuit today

### `.checks.yaml`

```yaml
checks:
  testSuitePassed:
    use: testSuitePassed
  codeQualityPassed:
    use: codeQualityPassed
```

### `check-registry.yaml`

```yaml
checks:
  testSuitePassed:
    kind: command
    command: make check
    returns: BOOL
  codeQualityPassed:
    kind: command
    command: make check-go-quality
    returns: BOOL
```

### `.prompts.yaml`

```yaml
states:
  spec:
    prompt: >
      Write the failing test for the task. Do not implement production code.
      When the targeted test is failing, request the transition event.
    event: writeTest
  red:
    prompt: >
      Implement the smallest code change that makes the failing test pass. Do
      not refactor or broaden scope. When the full check passes, request the
      transition event.
    event: implement
  green:
    prompt: >
      The slice is green. Request the transition event to run code quality
      review before deciding finish or refactor.
    event: reviewQuality
  qualityReview:
    prompt: >
      Circuit has checked code quality. If the quality gate passes, request
      finish. If the quality gate fails, request refactor.
  refactoring:
    prompt: >
      Refactor only to satisfy the code quality gate while preserving behavior.
      Use standard refactorings when they apply: Extract Function, Extract Type,
      Extract Helper, Rename for clarity, consolidate duplicate conditionals,
      and move repeated mapping logic into one place. When tests pass again,
      request the transition event.
    event: keepGreen
```

## What the driver adds at run time

The user only supplies:

```text
machine
task string
```

Circuit supplies at each turn:

- current state
- enabled and blocked operations
- session id
- state guidance (goal + prompt + expected event) from the prompts file
- the overall task string

The prompt Pi sees combines Circuit status output, the overall task, and the
state-specific prompt.

## Response contract

The agent backend is expected to respond with exactly one transition event
name. Circuit extracts the requested event and validates it against the
machine. Circuit runs checks and either accepts or blocks the transition. When
blocked, Circuit re-prompts the same state up to a bounded number of attempts.

Pi does not decide progress. Circuit does.

## Evidence

Session-scoped paths are the standard. For `tdd-flow` and any similar workflow:

```text
.tmp/circuit/<session-id>/
```

Check scripts must not read global evidence paths. Multiple concurrent sessions
must not share evidence. Circuit passes:

```text
CIRCUIT_SESSION_ID
CIRCUIT_MACHINE_NAME
CIRCUIT_MACHINE_FILE
CIRCUIT_CURRENT_STATE
```

into every check command's environment.

## Trace

`circuit drive` writes:

```text
.tmp/circuit/<session-id>/drive.jsonl
```

Trace events:

- `prompt`: text sent to the backend
- `response`: text returned by the backend
- `workspace`: git status short after the response
- `advance`: transition result

The trace is what makes dogfood evaluable. Without it we could only see final
diffs and final check states, which was not enough to distinguish real TDD from
production-code changes that happened to fail `make check`.

## What Circuit itself does not know

Circuit does not know:

- which specific test file corresponds to a slice
- which tools count as "code quality"
- what standard refactorings apply in a language
- what the human considers done

Those live in project-local scripts, the check registry, the prompt file, and
the task string.

## Substitution boundaries

Any project or workflow can substitute its own implementations, as long as the
contracts hold:

- A B machine that references a BOOL variable requires a check binding for it.
- A check binding requires a registry entry that returns BOOL.
- A check command returns 0 for TRUE and non-zero for FALSE.
- A prompt file maps states to prompts and optionally to expected event names.
- An agent backend implements `Prompt(message string) (response string, err)`.

Different repositories may:

- ship different machines
- point `testSuitePassed` at any command that returns BOOL
- point `codeQualityPassed` at any command that returns BOOL
- provide different prompt files for the same machine
- provide different agent backends (real Pi, fake Pi, other CLI agents)

The Circuit core does not change.

## What dogfood confirmed

- The user only needs `machine + task`.
- Circuit runs the workflow.
- Pi does the work.
- Checks decide truth.
- Refactoring becomes real when it is required to satisfy `codeQualityPassed`.
- Traces are needed to see what actually happened.
- Environment issues (like missing `.pi/node_modules`) can silently corrupt a
  run; smoke workflows must include workspace preparation, not only building
  the binary.

## What is still open

- A first-class prompt override mechanism (currently done by editing the prompt
  file in a worktree).
- Structured/typed trace events instead of ad-hoc JSON maps.
- Optional stricter red proof beyond `not(testSuitePassed = TRUE)`.
- Outer Pi UX (`/circuit:tdd <slice>`) wrapping `circuit drive tdd-flow`.
- More Go quality tools once the initial three prove stable.
