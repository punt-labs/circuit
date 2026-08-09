package circuitb

import (
	"strings"
	"testing"
)

func mustLex(t *testing.T, input string) []token {
	t.Helper()
	tokens, err := lex("test.mch", input)
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	return tokens
}

func mustParse(t *testing.T, input string) rawMachine {
	t.Helper()
	raw, err := parse(mustLex(t, input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return raw
}

func TestParseMinimalMachine(t *testing.T) {
	t.Parallel()
	raw := mustParse(t, `
		MACHINE Simple
		SETS STATE = {a, b}; TRANSITION = {go}
		VARIABLES current
		INVARIANT current : STATE
		INITIALISATION current := a
		OPERATIONS
			Advance(evt) = PRE evt : TRANSITION & current = a & evt = go THEN current := b END
		END
	`)
	if raw.Name != "Simple" {
		t.Fatalf("name = %q", raw.Name)
	}
	if len(raw.Sets) != 2 {
		t.Fatalf("sets = %d", len(raw.Sets))
	}
	if len(raw.Variables) != 1 || raw.Variables[0].Name != "current" {
		t.Fatalf("variables = %v", raw.Variables)
	}
	if len(raw.Operations) != 1 || raw.Operations[0].Name != "Advance" {
		t.Fatalf("operations = %v", raw.Operations)
	}
}

func TestParseMissingMachineKeyword(t *testing.T) {
	t.Parallel()
	_, err := parse(mustLex(t, "SETS STATE = {a}"))
	if err == nil || !strings.Contains(err.Error(), "MACHINE") {
		t.Fatalf("error = %v, want MACHINE", err)
	}
}

func TestParseMissingEnd(t *testing.T) {
	t.Parallel()
	_, err := parse(mustLex(t, `
		MACHINE NoEnd
		SETS STATE = {a}; TRANSITION = {go}
		VARIABLES current
		INVARIANT current : STATE
		INITIALISATION current := a
		OPERATIONS
			Op(evt) = PRE evt : TRANSITION THEN current := a END
	`))
	if err == nil {
		t.Fatal("missing END returned nil error")
	}
}

func TestParseMissingOperationEnd(t *testing.T) {
	t.Parallel()
	_, err := parse(mustLex(t, `
		MACHINE BadOp
		SETS STATE = {a}
		VARIABLES current
		INVARIANT current : STATE
		INITIALISATION current := a
		OPERATIONS
			Op(x) = PRE x : STATE THEN current := a
		END
	`))
	if err == nil {
		t.Fatal("missing operation END returned nil error")
	}
}

func TestParseParallelAssignment(t *testing.T) {
	t.Parallel()
	raw := mustParse(t, `
		MACHINE Par
		SETS STATE = {a, b}
		VARIABLES x, y
		INVARIANT x : STATE & y : STATE
		INITIALISATION x := a || y := b
		OPERATIONS
			Op(evt) = PRE evt : STATE THEN x := b || y := a END
		END
	`)
	if len(raw.Variables) != 2 {
		t.Fatalf("variables = %d", len(raw.Variables))
	}
}

func TestParseIfElsifElse(t *testing.T) {
	t.Parallel()
	raw := mustParse(t, `
		MACHINE Branch
		SETS STATE = {a, b, c}; TRANSITION = {go}
		VARIABLES current
		INVARIANT current : STATE
		INITIALISATION current := a
		OPERATIONS
			Advance(evt) = PRE evt : TRANSITION THEN
				IF current = a & evt = go THEN current := b
				ELSIF current = b & evt = go THEN current := c
				ELSE current := a
				END
			END
		END
	`)
	if len(raw.Operations) != 1 {
		t.Fatalf("operations = %d", len(raw.Operations))
	}
}

func TestParseDisjunction(t *testing.T) {
	t.Parallel()
	raw := mustParse(t, `
		MACHINE Disj
		SETS STATE = {a, b}; TRANSITION = {go}
		VARIABLES current
		INVARIANT current : STATE
		INITIALISATION current := a
		OPERATIONS
			Advance(evt) = PRE evt : TRANSITION & (current = a or current = b) THEN current := b END
		END
	`)
	if raw.Operations[0].Pre == nil {
		t.Fatal("operation has nil precondition")
	}
}

func TestParseSetLiteral(t *testing.T) {
	t.Parallel()
	raw := mustParse(t, `
		MACHINE SetLit
		SETS STATE = {a, b, c}; TRANSITION = {go}
		VARIABLES current
		INVARIANT current : STATE
		INITIALISATION current := a
		OPERATIONS
			Advance(evt) = PRE evt : TRANSITION & current : {a, b} THEN current := c END
		END
	`)
	if raw.Operations[0].Pre == nil {
		t.Fatal("operation has nil precondition")
	}
}

func TestParseComparisonOperators(t *testing.T) {
	t.Parallel()
	for _, op := range []string{"=", "/=", "<", ">", "<=", ">="} {
		raw := mustParse(t, `
			MACHINE Cmp
			SETS STATE = {a}
			VARIABLES current, count
			INVARIANT current : STATE & count : {0, 1, 2}
			INITIALISATION current := a || count := 0
			OPERATIONS
				Op(evt) = PRE evt : STATE & count `+op+` 1 THEN count := 1 END
			END
		`)
		if raw.Operations[0].Pre == nil {
			t.Fatalf("operator %s: nil precondition", op)
		}
	}
}

func TestParseMultipleOperations(t *testing.T) {
	t.Parallel()
	raw := mustParse(t, `
		MACHINE Multi
		SETS STATE = {a, b}; TRANSITION = {go, back}
		VARIABLES current
		INVARIANT current : STATE
		INITIALISATION current := a
		OPERATIONS
			Go(evt) = PRE evt : TRANSITION & evt = go THEN current := b END;
			Back(evt) = PRE evt : TRANSITION & evt = back THEN current := a END
		END
	`)
	if len(raw.Operations) != 2 {
		t.Fatalf("operations = %d, want 2", len(raw.Operations))
	}
	if raw.Operations[0].Name != "Go" || raw.Operations[1].Name != "Back" {
		t.Fatalf("operations = %s, %s", raw.Operations[0].Name, raw.Operations[1].Name)
	}
}

func TestParseOperationWithReturnValue(t *testing.T) {
	t.Parallel()
	raw := mustParse(t, `
		MACHINE Ret
		SETS STATE = {a}
		VARIABLES current
		INVARIANT current : STATE
		INITIALISATION current := a
		OPERATIONS
			result <-- Query(x) = PRE x : STATE THEN result := a END
		END
	`)
	if len(raw.Operations) != 1 || raw.Operations[0].Name != "Query" {
		t.Fatalf("operation = %v", raw.Operations)
	}
}

func TestParseUnsupportedANY(t *testing.T) {
	t.Parallel()
	raw := mustParse(t, `
		MACHINE WithAny
		SETS STATE = {a, b}; TRANSITION = {go}
		VARIABLES current
		INVARIANT current : STATE
		INITIALISATION current := a
		OPERATIONS
			Op(evt) = PRE evt : TRANSITION THEN ANY xx WHERE xx : STATE THEN current := xx END END
		END
	`)
	_ = raw
}

func TestParseBadPredicateMissingOperand(t *testing.T) {
	t.Parallel()
	_, err := parse(mustLex(t, `
		MACHINE Bad
		SETS STATE = {a}
		VARIABLES current
		INVARIANT current : STATE
		INITIALISATION current := a
		OPERATIONS
			Op(evt) = PRE & THEN current := a END
		END
	`))
	if err == nil {
		t.Fatal("bad predicate returned nil error")
	}
}

func TestParseBadAssignmentMissingValue(t *testing.T) {
	t.Parallel()
	_, err := parse(mustLex(t, `
		MACHINE Bad
		SETS STATE = {a}
		VARIABLES current
		INVARIANT current := 
		INITIALISATION current := a
		END
	`))
	if err == nil {
		t.Fatal("bad assignment returned nil error")
	}
}

func TestParseEmptyInput(t *testing.T) {
	t.Parallel()
	_, err := parse(mustLex(t, ""))
	if err == nil {
		t.Fatal("empty input returned nil error")
	}
}
