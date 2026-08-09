package circuitb

import (
	"strings"
	"testing"
)

func mustResolve(t *testing.T, input string) Machine {
	t.Helper()
	tokens, err := lex("test.mch", input)
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	raw, err := parse(tokens)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	machine, err := resolve(raw)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	return machine
}

func TestResolveEnumVariable(t *testing.T) {
	t.Parallel()
	machine := mustResolve(t, `
		MACHINE M
		SETS STATE = {a, b}; TRANSITION = {go}
		VARIABLES current
		INVARIANT current : STATE
		INITIALISATION current := a
		OPERATIONS
			Advance(evt) = PRE evt : TRANSITION & current = a & evt = go THEN current := b END
		END
	`)
	v, ok := machine.variables["current"]
	if !ok || v.kind != valueEnum || v.set != "STATE" {
		t.Fatalf("current variable = %#v", v)
	}
}

func TestResolveBoolVariable(t *testing.T) {
	t.Parallel()
	machine := mustResolve(t, `
		MACHINE M
		SETS STATE = {a}; TRANSITION = {go}
		VARIABLES current, flag
		INVARIANT current : STATE & flag : BOOL
		INITIALISATION current := a || flag := FALSE
		OPERATIONS
			Advance(evt) = PRE evt : TRANSITION & flag = TRUE THEN current := a END
		END
	`)
	v, ok := machine.variables["flag"]
	if !ok || v.kind != valueBool {
		t.Fatalf("flag variable = %#v", v)
	}
	if machine.initial["flag"].bool != false {
		t.Fatal("flag initial = true, want false")
	}
}

func TestResolveNatVariable(t *testing.T) {
	t.Parallel()
	machine := mustResolve(t, `
		MACHINE M
		SETS STATE = {a}; TRANSITION = {go}
		VARIABLES current, count
		INVARIANT current : STATE & count : NAT
		INITIALISATION current := a || count := 0
		OPERATIONS
			Advance(evt) = PRE evt : TRANSITION & count = 0 THEN count := 1 END
		END
	`)
	v, ok := machine.variables["count"]
	if !ok || v.kind != valueNat {
		t.Fatalf("count variable = %#v", v)
	}
	if machine.initial["count"].nat != 0 {
		t.Fatal("count initial != 0")
	}
}

func TestResolveMissingCurrentVariableFails(t *testing.T) {
	t.Parallel()
	tokens, _ := lex("test.mch", `
		MACHINE M
		SETS STATE = {a}; TRANSITION = {go}
		VARIABLES flag
		INVARIANT flag : BOOL
		INITIALISATION flag := FALSE
		OPERATIONS
			Op(evt) = PRE evt : TRANSITION THEN flag := TRUE END
		END
	`)
	raw, _ := parse(tokens)
	_, err := resolve(raw)
	if err == nil || !strings.Contains(err.Error(), "current") {
		t.Fatalf("resolve error = %v, want current variable", err)
	}
}

func TestResolveUnknownInitialisationVariable(t *testing.T) {
	t.Parallel()
	tokens, _ := lex("test.mch", `
		MACHINE M
		SETS STATE = {a}; TRANSITION = {go}
		VARIABLES current
		INVARIANT current : STATE
		INITIALISATION current := a || bogus := a
		OPERATIONS
			Op(evt) = PRE evt : TRANSITION THEN current := a END
		END
	`)
	raw, _ := parse(tokens)
	_, err := resolve(raw)
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("resolve error = %v, want unknown variable", err)
	}
}

func TestResolveSetsAndEnumValues(t *testing.T) {
	t.Parallel()
	machine := mustResolve(t, `
		MACHINE M
		SETS COLOR = {red, green, blue}; TRANSITION = {go}
		VARIABLES current
		INVARIANT current : COLOR
		INITIALISATION current := red
		OPERATIONS
			Op(evt) = PRE evt : TRANSITION THEN current := green END
		END
	`)
	colors, ok := machine.sets["COLOR"]
	if !ok || len(colors) != 3 {
		t.Fatalf("COLOR set = %v", colors)
	}
}

func TestResolveOperationParameters(t *testing.T) {
	t.Parallel()
	machine := mustResolve(t, `
		MACHINE M
		SETS STATE = {a, b}; TRANSITION = {go}
		VARIABLES current
		INVARIANT current : STATE
		INITIALISATION current := a
		OPERATIONS
			Advance(evt) = PRE evt : TRANSITION & current = a & evt = go THEN current := b END
		END
	`)
	if len(machine.operations) != 1 {
		t.Fatalf("operations = %d", len(machine.operations))
	}
	if len(machine.operations[0].Parameters) != 1 || machine.operations[0].Parameters[0] != "evt" {
		t.Fatalf("parameters = %v", machine.operations[0].Parameters)
	}
}

func TestResolveAdvanceOperation(t *testing.T) {
	t.Parallel()
	machine := mustResolve(t, `
		MACHINE M
		SETS STATE = {a, b}; TRANSITION = {go}
		VARIABLES current
		INVARIANT current : STATE
		INITIALISATION current := a
		OPERATIONS
			Advance(evt) = PRE evt : TRANSITION THEN current := b END
		END
	`)
	if machine.advance.Name != "Advance" {
		t.Fatalf("advance operation = %q", machine.advance.Name)
	}
}
