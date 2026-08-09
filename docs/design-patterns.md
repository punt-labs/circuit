# Design Patterns

## Naming

Excellent names are the first rule of good software design.

Kent Beck's Smalltalk Best Practice Patterns treats naming as design, not
cosmetics. The pattern is simple: name things from the reader's point of view so
that the name reveals the object's role and the sender's intent. A good name
removes the need to inspect the implementation before understanding the design.

Use names that say what concept the code represents:

- `machines/` is better than `specs/` when the files are runtime machines.
- `internal/circuitb/` is better than `internal/bprofile/` when the package is
  the Circuit-supported B language.
- `circuit status` is better than `circuit state` when the command reports the
  active circuit, current B state, enabled transitions, blocked transitions, and
  runtime check metadata.

Names are part of the model. If the name is wrong, the abstraction is already
leaking.

## Verification

> Any property of software not verified, does not exist.
>
> — Ralph Johnson

For `circuit`, this means a state-machine property is not real because the
README says it, because the B machine suggests it, or because the adapter assumes
it. It exists only when the relevant gate verifies it:

- ProB verifies the B machine during development and release.
- Go tests verify the Circuit-B runtime interpretation.
- Harness spikes verify that pi/opencode/Claude adapters obey the machine rather
  than reimplementing its transition logic.
