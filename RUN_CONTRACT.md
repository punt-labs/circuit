# Run contract sketch

This is a working note, not a settled ADR.

The current intuition is that Circuit has a pure machine and local checks, but
when Circuit drives an agent it also needs a run contract. The run contract is
not the machine. The machine is still the formal source of truth for states,
transitions, and guards. The contract tells Circuit what concrete work should
happen while the machine is in a state.

A useful way to think about it:

```text
B machine:
  what states exist
  what transitions exist
  what facts must hold for transitions

checks.yaml / check-registry.yaml:
  how external boolean facts are observed

run contract:
  what goal/prompt applies while in each state
  what the agent should do before requesting the next transition
```

So the contract may be parallel to check bindings. Checks attach outside-world
truth to B variables. State prompts attach outside-world work instructions to B
states.

## Possible shape

A contract might look roughly like this:

```yaml
machine: tdd-flow
name: tdd-slice

goal: >
  Deliver one TDD slice for a concrete product task. The machine owns the
  workflow; the agent may only request progress through enabled transitions.

stateGoals:
  spec: >
    Establish the failing test for this slice.
  red: >
    Make the failing test pass with the smallest implementation.
  green: >
    Decide whether the slice is done or needs refactoring.
  refactoring: >
    Improve the implementation while keeping checks green.

statePrompts:
  spec: >
    You are in the spec phase. Write the failing test for the slice and create
    the session-scoped red evidence file. Do not implement production code.
    When the red test is proven, request event writeTest.

  red: >
    You are in the red phase. Implement the smallest code change that makes the
    failing test pass. Do not refactor or broaden scope. When the suite is
    green, request event implement.

  green: >
    You are in the green phase. If the slice is complete, request event finish.
    If cleanup is needed, request event refactor.

  refactoring: >
    You are in the refactoring phase. Improve structure without changing
    behavior. When checks are green, request event keepGreen.

responseContract:
  format: event-name
  rule: >
    At the end of the turn, respond with exactly one transition event name that
    Circuit can validate. Do not claim progress outside a successful transition.

evidence:
  baseDir: .tmp/circuit/${sessionId}
  suiteStamp: .tmp/circuit/${sessionId}/suite-green.stamp

failurePolicy:
  onNoEvent: reprompt
  onBlocked: reprompt
  maxAttemptsPerState: 3
```

This is only a sketch. The important idea is that the contract gives Circuit a
state-indexed instruction layer without changing the B machine.

## Goal and prompt per state

The user suggestion seems right: each state probably wants both a goal and a
prompt.

The state goal is stable and short:

```text
spec goal: establish a failing test
red goal: make it pass
green goal: decide done or refactor
refactoring goal: clean up while green
```

The state prompt is operational and may include the concrete task, evidence
paths, allowed edits, and response instructions.

This mirrors checks:

```text
B boolean variable -> bound check command
B state            -> bound goal/prompt
```

Checks answer: what must be true for this transition?

State prompts answer: what should the driven agent do while we are here?

## When prompts appear

A state prompt should appear whenever Circuit enters or remains in a state and
is about to drive an agent turn.

For example:

```text
start tdd-flow -> current state spec
Circuit emits/uses spec goal + spec prompt
Pi does spec work
Pi requests writeTest
Circuit checks testSuitePassed (via not(...) in tdd-flow)
Circuit advances spec -> red
Circuit emits/uses red goal + red prompt
```

If Pi proposes a blocked event or no event, Circuit stays in the same state and
reuses the same state's prompt, probably augmented with the failure reason.

So a state prompt is not a transition action in B. It is a driver-side behavior
triggered by the current state.

## Closed and open parts

The closed parts should be:

```text
B evaluator
session lifecycle
check invocation semantics
driver loop skeleton
backend prompt/response transport
```

These should not change when we add TDD, PR review, release, or other workflows.

The open parts should be:

```text
machines
run contracts
state goals
state prompts
response extractors
agent backends
check implementations
evidence formats
failure policies
approval policies
```

That is the open/closed split.

## Liskov-style substitution

The driver should depend on small behavioral interfaces.

An agent backend is substitutable if:

```text
Prompt(string) -> response string or transport error
it does not mutate Circuit state directly
it does not decide progress
```

A prompt strategy is substitutable if:

```text
status + contract -> prompt
it reflects the current machine state
it does not invent transitions
```

A response extractor is substitutable if:

```text
response + status + contract -> requested event or no event
it does not turn arbitrary prose into progress
```

A check implementation is substitutable if:

```text
exit 0 means TRUE
nonzero means FALSE
session-specific evidence is scoped by session id
```

The driver loop can then stay generic.

## Driver loop sketch

```text
for session while active:
  status = circuit.status(session)
  contractState = contract.state[status.current]
  prompt = render(contract.goal, contractState.goal, contractState.prompt, status)
  response = backend.prompt(prompt)
  event = extractor.extract(response, status, contract)
  if no event:
    reprompt according to failurePolicy
    continue
  result = circuit.advance(event, session)
  if blocked:
    reprompt with blocked reason
    continue
  if terminal:
    stop
```

This is the part that should become closed once designed.

## How this relates to Pi -> Circuit -> Pi

Outer Pi or the user supplies the slice:

```text
/circuit:tdd "Add load --json"
```

Circuit starts or attaches to the machine session and loads the run contract.

Circuit then drives inner Pi with the state prompt. Inner Pi does work and
requests transitions. Circuit validates. Inner Pi does not decide truth.

That is the direction that seems most on track right now.

## Open questions

- Is the contract keyed only by state, or by state plus transition?
- Does each state have one prompt, or multiple possible prompts based on enabled
  operations?
- Should response format be plain event names first, or structured JSON from the
  start?
- How much tool policy belongs in the contract versus in the backend launcher?
- Should check evidence conventions live in the run contract, or only in local
  check scripts?
- How does the outer human approve or interrupt between states?
