# B Machines for Circuit

## Status

Design note. This is the current working direction for making `circuit` a
formal, locally hosted state-machine guide for agent harnesses.

## Thesis

Circuit workflow definitions should be B abstract machines.

The B machine is the authored artifact, the reviewed artifact, and the runtime
artifact. ProB is required for development and release validation, but not for
normal circuit use.

```text
machines/pr-watch.mch
  ├─ development/release: probcli proves and model-checks the machine
  └─ runtime: Go parses and evaluates the supported Circuit-B profile
```

Harnesses such as pi, Claude Code, and opencode do not define workflow
semantics. They observe facts, present status, and request operations against the
machine. The machine is the source of truth for valid progress.

## Spike roadmap

Three spikes define the current direction:

1. **B-machine runtime.** Prove Go can parse and evaluate a strict Circuit-B
   profile from the same `.mch` files that ProB checks during development.
2. **Pi hosts Circuit.** Prove pi can start an active circuit, report status,
   and request transitions through the Go runtime without duplicating machine
   logic.
3. **Circuit drives pi.** Test Circuit as the outer runner that owns machine
   state and drives `pi --mode rpc` as an agent backend.

Spike 1 and the pi-hosted part of spike 2 now have working code. The next
useful proof is richer check binding behavior and then the circuit-driven pi RPC
spike.

## Motivation

Circuit is not just a configuration format for workflows. It is intended to
constrain an agent harness to a valid state-machine path:

- the current state is known;
- valid next operations are known;
- operation preconditions are explicit;
- invalid transitions are rejected;
- release candidates can be proven or model-checked before use.

B is a strong fit because an agent workflow maps directly onto B concepts.

| Circuit concept | B concept |
| --- | --- |
| workflow definition | `MACHINE` |
| states, events, domains | enumerated `SETS` |
| current workflow position | `VARIABLES current` |
| facts/evidence | additional `VARIABLES` |
| type and safety constraints | `INVARIANT` |
| starting state | `INITIALISATION` |
| transition | `OPERATION` |
| guard condition | `PRE` |
| state update | substitution after `THEN` |
| release validation | ProB parse, animation, model checking, refinement |

This preserves readability. A human reviews a transition like:

```b
ReadyToMerge =
    PRE
        current = watch &
        checks = success &
        unresolvedThreads = 0 &
        materialFindings = 0
    THEN
        current := merging
    END
```

not a generated JSON predicate tree.

## Development-time and runtime split

Circuit development can require ProB. Circuit runtime should not.

Development and release use ProB to validate the formal B machine:

```bash
probcli machines/pr-watch.mch -init
probcli machines/pr-watch.mch -animate 20
probcli machines/pr-watch.mch -cbc_assertions
probcli machines/pr-watch.mch -model_check
```

Runtime uses the Go implementation to parse and evaluate the same `.mch` file.
The user-facing machine remains B; there is no required generated JSON artifact.
A normalized JSON view may exist later for debugging, caching, or embedding, but
it is not the primary authoring format.

Release eligibility for a machine should require both:

1. ProB accepts and model-checks the machine within the agreed bounds.
2. The Go runtime accepts the machine as valid Circuit-B and evaluates it
   according to the supported profile.

## Circuit-B profile

Circuit does not interpret all B. It interprets a strict B profile designed for
finite state machines.

Valid B outside this profile may still be useful for proof and refinement, but
`circuit run` must reject unsupported constructs with a clear diagnostic.

### Initial supported structure

The first profile should support this shape:

```b
MACHINE BuildJob
SETS
    STATE = {idle, running, done};
    TRANSITION = {start, finish}
VARIABLES
    current
INVARIANT
    current : STATE
INITIALISATION
    current := idle
OPERATIONS
    Advance(evt) =
        PRE
            evt : TRANSITION &
            current /= done &
            (
                (current = idle & evt = start) or
                (current = running & evt = finish)
            )
        THEN
            IF current = idle & evt = start THEN
                current := running
            ELSIF current = running & evt = finish THEN
                current := done
            END
        END
END
```

This exact style has already been smoke-tested with ProB. ProB initialized and
model-checked it, reporting three analysed states, all open states visited, all
operations covered, and no counterexample.

### Supported declarations

Initial support:

- `MACHINE Name`
- enumerated `SETS`
- scalar `VARIABLES`
- `INVARIANT` type declarations over supported domains
- deterministic `INITIALISATION`
- `OPERATIONS`
- `PRE ... THEN ... END`
- `IF ... THEN ... ELSIF ... ELSE ... END`
- scalar assignment
- parallel assignment with B simultaneous semantics

Initial scalar domains:

- enumerated sets
- `NAT`
- `BOOL`

`INT` remains future work.

### Supported predicates

Initial predicate grammar:

```text
predicate    := disjunction
disjunction := conjunction ("or" conjunction)*
conjunction := atom ("&" atom)*
atom        := negation | comparison | membership | "(" predicate ")"
negation    := "not" "(" predicate ")"
comparison  := expr ("=" | "/=" | "<" | "<=" | ">" | ">=") expr
membership  := expr ":" set_expr
set_expr    := identifier | "{" expr ("," expr)* "}"
expr        := identifier | integer
```

This is enough for guards such as:

```b
checks : {failure, cancelled, timedOut} or materialFindings > 0
```

and:

```b
checks = success & unresolvedThreads = 0 & materialFindings = 0
```

### Unsupported initially

The runtime should reject these at profile-validation time:

- quantifiers;
- set comprehensions;
- arbitrary relations and functions;
- sequences;
- nondeterministic substitution (`ANY`, `CHOICE`, `SELECT`);
- loops;
- imports/includes;
- refinements as runtime artifacts;
- machine composition;
- operation bodies that mutate multiple variables in unsupported ways;
- expressions requiring general B evaluation.

ProB may accept these constructs. Circuit runtime still rejects them until the
profile is explicitly expanded.

## Multi-pass implementation

The parser should be treated as a small compiler component. A single-pass design
is likely to accumulate context flags and hidden coupling as the profile grows.
The txt2tex experience suggests using multiple passes from the start.

### Pass 1: lex and parse structure

Input: `.mch` text.

Output: raw AST with source spans.

Responsibilities:

- tokenize keywords, identifiers, literals, operators, and punctuation;
- parse machine clauses;
- identify clause boundaries;
- parse operation declarations;
- parse supported predicate and substitution syntax;
- attach source spans to every token and AST node.

This pass should not resolve whether an identifier is a variable, enum value, set
name, or parameter. It should parse structure only.

### Pass 2: resolve names and types

Input: raw AST.

Output: resolved AST or diagnostics.

Responsibilities:

- build symbol tables from `SETS`, `VARIABLES`, operation parameters, and
  invariant type declarations;
- resolve identifiers to variables, sets, enum values, or parameters;
- infer/check scalar expression types within the supported subset;
- detect unknown identifiers and type mismatches;
- annotate expressions with resolved symbols where useful for evaluation.

This pass avoids parser context flags. The parser does not guess what a name
means; name meaning is decided after the whole structure is known.

### Pass 3: validate Circuit-B profile

Input: resolved AST.

Output: executable Circuit-B machine or diagnostics.

Responsibilities:

- reject valid-B-but-unsupported constructs;
- enforce the finite-state-machine shape;
- ensure state variables and transitions are recognizable;
- ensure terminal states are explicit or derivable;
- ensure substitutions are deterministic and evaluable by the Go runtime;
- produce profile errors that explain how to rewrite unsupported B.

### Pass 4: evaluate

Input: profile-valid machine plus runtime values.

Output: active circuit status, enabled operations, blocked operations, or
updated state.

Responsibilities:

- evaluate preconditions over scalar runtime values;
- report which operation preconditions are satisfied;
- apply substitutions with B semantics;
- preserve frame conditions for unassigned variables;
- apply parallel assignments simultaneously, not sequentially.

## Diagnostics

Good diagnostics are required. They are not polish.

Every token and AST node should carry a source span:

```go
type Span struct {
    File      string
    Line      int
    Column    int
    EndLine   int
    EndColumn int
}
```

Diagnostics should distinguish three categories.

### Syntax errors

The file does not match the supported grammar.

Example:

```text
machines/pr-watch.mch:14:5: expected THEN after PRE block, found END
```

### Name and type errors

The syntax is valid but names or types do not resolve.

Example:

```text
machines/pr-watch.mch:18:9: unknown identifier chekcs in operation NeedsWork
  hint: did you mean checks?
```

Example:

```text
machines/pr-watch.mch:22:9: type mismatch: materialFindings is NAT;
cannot compare with enum value success
```

### Profile violations

The B may be valid, but Circuit runtime does not support it.

Example:

```text
machines/pr-watch.mch:25:5: unsupported B construct in operation Observe
  ANY substitution is not supported by Circuit runtime
  supported substitutions: assignment, parallel assignment, IF/ELSIF, PRE/THEN
```

Profile diagnostics should explicitly say that ProB may accept the construct but
Circuit cannot execute it.

### Recovery strategy

The parser does not need language-server-quality recovery. It should recover at
coarse boundaries when possible:

- next clause keyword;
- next operation declaration;
- matching `END`.

Name/type/profile passes should accumulate diagnostics rather than stopping at
the first issue.

## Terminal states and deadlock

A terminal state intentionally has no outgoing workflow transition. Therefore a
blanket deadlock-freedom requirement is not the right property.

Circuit should distinguish acceptable terminal deadlock from unexpected workflow
stalls. The B machine should make terminal states explicit or derivable, and the
release checks should verify the intended property:

```text
No non-terminal state is stuck under the intended fact model.
```

The exact ProB assertion or checking pattern still needs a spike. It should not
be reduced to blindly running `-cbc_deadlock` and treating every terminal state
as a failure.

## Check bindings

Some transition preconditions depend on the outside world. For example,
advancing from `coding` to `codeReview` may require the local quality gate to
pass.

B should model the required boolean, not the shell command:

```b
makeCheckPassed = TRUE
```

The command binding belongs beside the machine:

```text
machines/review-flow.mch
machines/review-flow.checks.yaml
machines/check-registry.yaml
```

`review-flow.checks.yaml` maps B variables to registered checks:

```yaml
checks:
  makeCheckPassed:
    use: makeCheck
```

`check-registry.yaml` defines the allowed checks:

```yaml
checks:
  makeCheck:
    kind: command
    command: make check
    returns: BOOL
```

Runtime behavior for `circuit advance requestReview`:

1. Load the active machine.
2. Load its check bindings and the check registry.
3. Run checks bound to B variables.
4. Store check booleans and invocation counts in the active Go run.
5. Evaluate the B precondition with those values.
6. Advance if allowed; otherwise block and report failed preconditions.

B remains pure and ProB-checkable. Commands remain outside B. Circuit validates
that every BOOL fact has a binding, every binding names a B variable, every
binding references a registry entry, and every registry entry returns a
compatible type.

`status` should report check state as runtime metadata, for example:

```text
checks:
  makeCheckPassed: TRUE (invocations: 1)
```

## Runtime lifecycle: sessions

Each circuit run is a session with explicit lifecycle states:

```text
SessionState ::= unloaded | active | suspended | stopped
```

- `unloaded` — no session selected or persisted.
- `active` — session is in memory, workflow state is progressing.
- `suspended` — session serialized under `.tmp/sessions/` between CLI calls.
- `stopped` — session reached a terminal state or was explicitly stopped.

Short-lived CLI invocations use implicit boundaries:

```text
suspended -> active -> suspended
```

When `Advance` reaches a terminal state (no enabled operations), that session
automatically transitions to `stopped`. Stopped sessions remain known so `stop`
is idempotent for known sessions; unloaded or unknown sessions cannot be
stopped. `Status` reports known active or stopped sessions; `Advance` requires
at least one active session. `Unload` removes stopped sessions from runtime
storage. `status` with no known session reports "no session" as informational
output, not an error. Implicit `advance` requires exactly one active session;
implicit `stop` requires exactly one active or stopped session; otherwise
callers target a session ID such as `build-job-a3f8`.

Context injection in pi fires only when at least one session is active and
includes every active session.

## Runtime commands

The user-facing Go CLI operates on active circuit sessions:

```bash
circuit list
circuit load pr-watch
circuit scaffold pr-watch
circuit start pr-watch
circuit status
circuit advance ReadyToMerge
circuit stop
circuit unload build-job-b4c9

# Optional session qualifiers disambiguate multiple active sessions.
circuit advance ReadyToMerge pr-watch-a3f8
circuit stop build-job-b4c9
```

`list` shows available B machines from `machines/*.mch`.

`load` validates the B machine, its BOOL check bindings, and registry entries
without starting a session.

`scaffold` generates missing `.checks.yaml` bindings and missing
`check-registry.yaml` entries for resolved BOOL variables. Generated commands are
`false`, so unimplemented checks block safely.

`start` loads the machine, initializes a new active session, and records the
machine plus current B state.

`status` reports all known active or stopped sessions, or one selected session.
State is the B-machine variable; status is the operational report. The first
status report includes:

- session ID;
- machine;
- current B-machine state;
- enabled operations;
- blocked operations.

Later status reports should grow operational metadata:

- start time;
- elapsed time;
- accepted transition count;
- blocked transition count;
- latest accepted transition;
- latest blocked transition and failed preconditions.

`advance <event> [session]` requests one explicit operation/event against the
only active session or the selected session. If multiple operations are enabled,
Circuit does not guess. The caller must choose. If the requested operation is
blocked, the command reports failed preconditions and leaves the active state
unchanged.

`stop [session]` stops the only active session or the selected session. Stopping
a known stopped session is an idempotent success; stopping an unloaded or unknown
session is not.

`unload <session>` removes a stopped session from runtime storage. Active
sessions cannot be unloaded.

## Harness relationships

B-backed runtime supports both candidate pi relationships.

### Pi hosts engine

Pi remains the interactive harness. The project-local pi extension calls the Go
runtime with the `.mch` file.

```text
pi extension
  -> circuit start pr-watch
  -> circuit status
  -> circuit advance ReadyToMerge
```

Pi owns UI, status widgets, session integration, and user interaction. The B
machine owns valid progress. The pi command surface should mirror the CLI:
`/circuit list`, `/circuit load <machine>`, `/circuit scaffold <machine>`,
`/circuit start <machine>`, `/circuit status`,
`/circuit advance <event> [session]`, `/circuit stop [session]`, and
`/circuit unload <session>`.

### Engine drives pi over RPC

Circuit is the outer runner. It owns the B machine state and uses pi RPC as an
agent backend.

```text
circuit runner
  -> asks pi to observe or act
  -> updates facts
  -> evaluates enabled operations
  -> sends next-state guidance to pi
```

This is better for headless or supervised automation. The spike
(`cmd/circuit-rpc-spike/`) demonstrated a full `idle -> running -> done` loop
where the Go runner owned the B-machine state, sent state-aware prompts to
`pi --mode rpc`, waited for `agent_settled`, extracted the agent's chosen
operation, and validated it against the B machine before advancing.

## Go package shape

Initial package layout:

```text
internal/circuitb/
  tokens.go
  lexer.go
  ast.go
  parser.go
  resolve.go
  profile.go
  eval.go
  diagnostics.go
```

Responsibilities:

- `lexer.go`: token stream with source positions;
- `parser.go`: raw AST construction;
- `resolve.go`: symbol tables and type resolution;
- `profile.go`: Circuit-B profile validation;
- `eval.go`: runtime precondition evaluation and substitutions;
- `diagnostics.go`: source spans, error levels, formatting.

The implementation should not mix parsing with evaluation.

## Lightweight ADRs: parser technology

### ADR 1: Hand-written parser for the first Circuit-B profile

Use a hand-written lexer and recursive-descent parser for the first Circuit-B
profile.

Rationale:

- Circuit-B is a small, strict profile, not full B.
- The parser must produce good source spans and diagnostics.
- The implementation needs profile errors that explain: "ProB may accept this,
  but Circuit runtime cannot execute it."
- The current risk is semantic drift and error quality, not raw grammar volume.

This decision is reversible. The public boundary is the `internal/circuitb`
package API, not the parser implementation.

### ADR 2: Parser libraries may replace pass 1 only

A parser library or generator can replace lexing/raw parsing, but it does not
replace the multi-pass architecture.

The invariant remains:

```text
lex/parse raw AST -> resolve names/types -> validate profile -> evaluate
```

Name resolution, type checking, Circuit-B profile validation, and runtime
evaluation stay explicit passes even if the raw parser changes.

### ADR 3: Participle is the first alternative to test

If the hand-written parser becomes awkward after `PRWatch.mch`, the first
comparison spike should be `participle`.

Evaluation criteria:

- Does the grammar become clearer?
- Are source spans good enough?
- Are syntax errors better or worse?
- Can unsupported B constructs still produce profile-specific diagnostics?
- Does the AST remain suitable for separate resolve/profile/eval passes?

`pigeon` is the next candidate if a grammar file becomes more readable than
struct tags. `goyacc`, ANTLR, and gocc are too heavy for the current profile.

### ADR 4: Tree-sitter is tooling, not runtime

Tree-sitter is not a runtime parser candidate for Circuit-B. It may become
useful later for editor tooling, syntax highlighting, navigation, or incremental
analysis of `.mch` files.

### ADR 5: Study Go and HCL parser design

The standard library `go/scanner`, `go/parser`, `go/ast`, and HashiCorp HCL are
useful design references for production-grade hand-written parsers, especially
for source positions, AST shape, error lists, and diagnostics.

## Development gates

Add a `check-machines` target once `machines/` exists.

Near-term direct ProB target:

```make
check-machines:
        probcli machines/build-job.mch -init
        probcli machines/build-job.mch -model_check
```

Later, when z-spec B commands are available:

```make
check-machines:
        z-spec b-animate machines/build-job.mch
```

The aggregate `make check` should include `check-machines` only when the dev shell
or repo toolchain guarantees ProB availability. Otherwise, use a separate target
and document that release validation requires it.

## Open questions

1. Should the preferred authored style use one generic `Advance(evt)` operation
   or one B operation per workflow transition?
2. How should terminal states be declared: naming convention, set, assertion, or
   derived from lack of outgoing transitions?
3. What exact ProB assertion pattern verifies no unexpected non-terminal stalls?
4. How much of B's type system should Circuit-B support beyond enum sets and
   `NAT`?
5. Which runtime metadata belongs in `status` first: start time, elapsed time,
   accepted/blocked transition counts, latest transition, or all of them?
6. Should Circuit later support `.ref` refinement files for implementation
   strategies, or keep refinements development-only?

## Recommendation

Proceed with a narrow Circuit-B parser and evaluator in Go.

Do not parse all B. Parse and execute a strict finite-state-machine profile.
Use ProB as the development and release oracle. Keep the `.mch` file as the
runtime artifact so that readability remains central.

The next implementation spike should use `build-job.mch` first, not pi. Once
`start`, `status`, and `advance` work against that machine, add a real
`PRWatch.mch` and then test both pi integration relationships.
