package circuitb

import "testing"

func loadTestMachine(t *testing.T, input string) Machine {
	t.Helper()
	tokens, err := lex("test.mch", input)
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	raw, err := parse(tokens)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := validateProfile(raw); err != nil {
		t.Fatalf("profile: %v", err)
	}
	machine, err := resolve(raw)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	return machine
}

func TestEvalStateReportsEnabledAndBlocked(t *testing.T) {
	t.Parallel()
	machine := loadTestMachine(t, `
		MACHINE M
		SETS STATE = {a, b}; TRANSITION = {go, back}
		VARIABLES current
		INVARIANT current : STATE
		INITIALISATION current := a
		OPERATIONS
			Advance(evt) = PRE evt : TRANSITION & (
				(current = a & evt = go) or
				(current = b & evt = back)
			) THEN
				IF current = a & evt = go THEN current := b
				ELSIF current = b & evt = back THEN current := a
				END
			END
		END
	`)
	report, err := machine.State(nil)
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if report.Current != "a" {
		t.Fatalf("current = %q", report.Current)
	}
	if !containsCall(report.Enabled, "Advance(go)") {
		t.Fatalf("enabled = %v, want Advance(go)", report.Enabled)
	}
	if !containsCall(report.Blocked, "Advance(back)") {
		t.Fatalf("blocked = %v, want Advance(back)", report.Blocked)
	}
}

func TestEvalAdvanceChangesState(t *testing.T) {
	t.Parallel()
	machine := loadTestMachine(t, `
		MACHINE M
		SETS STATE = {a, b}; TRANSITION = {go}
		VARIABLES current
		INVARIANT current : STATE
		INITIALISATION current := a
		OPERATIONS
			Advance(evt) = PRE evt : TRANSITION & current = a & evt = go THEN current := b END
		END
	`)
	result, err := machine.Advance("go", nil)
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if !result.Allowed || result.To != "b" {
		t.Fatalf("advance = %#v", result)
	}
}

func TestEvalAdvanceBlockedReportsFailedPredicates(t *testing.T) {
	t.Parallel()
	machine := loadTestMachine(t, `
		MACHINE M
		SETS STATE = {a, b}; TRANSITION = {go}
		VARIABLES current
		INVARIANT current : STATE
		INITIALISATION current := a
		OPERATIONS
			Advance(evt) = PRE evt : TRANSITION & current = b & evt = go THEN current := a END
		END
	`)
	result, err := machine.Advance("go", nil)
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if result.Allowed {
		t.Fatal("advance should be blocked")
	}
	if len(result.Failed) == 0 {
		t.Fatal("blocked advance has no failed predicates")
	}
}

func TestEvalComparisonOperators(t *testing.T) {
	t.Parallel()
	cases := []struct {
		op    string
		left  int
		right int
		want  bool
	}{
		{"=", 1, 1, true},
		{"=", 1, 2, false},
		{"/=", 1, 2, true},
		{"/=", 1, 1, false},
		{"<", 1, 2, true},
		{"<", 2, 1, false},
		{">", 2, 1, true},
		{">", 1, 2, false},
		{"<=", 1, 1, true},
		{"<=", 1, 2, true},
		{"<=", 2, 1, false},
		{">=", 1, 1, true},
		{">=", 2, 1, true},
		{">=", 1, 2, false},
	}
	for _, c := range cases {
		pred := comparisonPredicate{
			operator: c.op,
			left:     numberExpression{value: c.left},
			right:    numberExpression{value: c.right},
		}
		got := pred.evaluate(nil, nil)
		if got != c.want {
			t.Errorf("%d %s %d = %v, want %v", c.left, c.op, c.right, got, c.want)
		}
	}
}

func TestEvalMembershipInNamedSet(t *testing.T) {
	t.Parallel()
	set := namedSetExpression{name: "STATE", values: []string{"a", "b"}}
	yes := set.contains(value{kind: valueEnum, enum: "a"}, nil, nil)
	no := set.contains(value{kind: valueEnum, enum: "c"}, nil, nil)
	if !yes {
		t.Error("a should be in STATE")
	}
	if no {
		t.Error("c should not be in STATE")
	}
}

func TestEvalMembershipInLiteralSet(t *testing.T) {
	t.Parallel()
	set := literalSetExpression{values: []expression{
		identifierExpression{name: "a"},
		identifierExpression{name: "b"},
	}}
	values := map[string]value{"a": {kind: valueEnum, enum: "a"}, "b": {kind: valueEnum, enum: "b"}}
	yes := set.contains(value{kind: valueEnum, enum: "a"}, values, nil)
	no := set.contains(value{kind: valueEnum, enum: "c"}, values, nil)
	if !yes {
		t.Error("a should be in literal set")
	}
	if no {
		t.Error("c should not be in literal set")
	}
}

func TestEvalMembershipBoolInBOOL(t *testing.T) {
	t.Parallel()
	set := namedSetExpression{name: "BOOL"}
	yes := set.contains(value{kind: valueBool, bool: true}, nil, nil)
	noNat := set.contains(value{kind: valueNat, nat: 1}, nil, nil)
	if !yes {
		t.Error("TRUE should be in BOOL")
	}
	if noNat {
		t.Error("NAT should not be in BOOL")
	}
}

func TestEvalMembershipNatInNAT(t *testing.T) {
	t.Parallel()
	set := namedSetExpression{name: "NAT"}
	yes := set.contains(value{kind: valueNat, nat: 42}, nil, nil)
	noEnum := set.contains(value{kind: valueEnum, enum: "a"}, nil, nil)
	if !yes {
		t.Error("42 should be in NAT")
	}
	if noEnum {
		t.Error("enum should not be in NAT")
	}
}

func TestEvalAssignment(t *testing.T) {
	t.Parallel()
	a := assignment{name: "x", value: numberExpression{value: 5}}
	result := a.apply(map[string]value{"x": {kind: valueNat, nat: 0}}, nil)
	if result["x"].nat != 5 {
		t.Fatalf("assignment x = %d, want 5", result["x"].nat)
	}
}

func TestEvalParallelAssignment(t *testing.T) {
	t.Parallel()
	pa := parallelAssignment{assignments: []assignment{
		{name: "x", value: numberExpression{value: 1}},
		{name: "y", value: numberExpression{value: 2}},
	}}
	result := pa.apply(map[string]value{
		"x": {kind: valueNat, nat: 0},
		"y": {kind: valueNat, nat: 0},
	}, nil)
	if result["x"].nat != 1 || result["y"].nat != 2 {
		t.Fatalf("parallel = x:%d y:%d", result["x"].nat, result["y"].nat)
	}
}

func TestEvalIfSubstitution(t *testing.T) {
	t.Parallel()
	sub := ifSubstitution{
		branches: []guardedSubstitution{
			{
				condition: comparisonPredicate{operator: "=", left: numberExpression{value: 1}, right: numberExpression{value: 1}},
				body:      assignment{name: "x", value: numberExpression{value: 10}},
			},
		},
	}
	result := sub.apply(map[string]value{"x": {kind: valueNat, nat: 0}}, nil)
	if result["x"].nat != 10 {
		t.Fatalf("if-sub x = %d, want 10", result["x"].nat)
	}
}

func TestEvalIfSubstitutionFallback(t *testing.T) {
	t.Parallel()
	sub := ifSubstitution{
		branches: []guardedSubstitution{
			{
				condition: comparisonPredicate{operator: "=", left: numberExpression{value: 1}, right: numberExpression{value: 2}},
				body:      assignment{name: "x", value: numberExpression{value: 10}},
			},
		},
		fallback: assignment{name: "x", value: numberExpression{value: 99}},
	}
	result := sub.apply(map[string]value{"x": {kind: valueNat, nat: 0}}, nil)
	if result["x"].nat != 99 {
		t.Fatalf("if-sub fallback x = %d, want 99", result["x"].nat)
	}
}

func TestEvalIfSubstitutionNoMatch(t *testing.T) {
	t.Parallel()
	sub := ifSubstitution{
		branches: []guardedSubstitution{
			{
				condition: comparisonPredicate{operator: "=", left: numberExpression{value: 1}, right: numberExpression{value: 2}},
				body:      assignment{name: "x", value: numberExpression{value: 10}},
			},
		},
	}
	values := map[string]value{"x": {kind: valueNat, nat: 0}}
	result := sub.apply(values, nil)
	if result["x"].nat != 0 {
		t.Fatalf("if-sub no-match x = %d, want 0", result["x"].nat)
	}
}

func TestEvalIdentifierLookupBindings(t *testing.T) {
	t.Parallel()
	expr := identifierExpression{name: "evt"}
	bindings := map[string]value{"evt": {kind: valueEnum, enum: "go"}}
	result := expr.evaluate(nil, bindings)
	if result.enum != "go" {
		t.Fatalf("binding lookup = %q, want go", result.enum)
	}
}

func TestEvalIdentifierLookupValues(t *testing.T) {
	t.Parallel()
	expr := identifierExpression{name: "current"}
	values := map[string]value{"current": {kind: valueEnum, enum: "idle"}}
	result := expr.evaluate(values, nil)
	if result.enum != "idle" {
		t.Fatalf("value lookup = %q, want idle", result.enum)
	}
}

func TestEvalExplainFormatsText(t *testing.T) {
	t.Parallel()
	pred := comparisonPredicate{
		operator: "=",
		left:     identifierExpression{name: "current"},
		right:    identifierExpression{name: "idle"},
	}
	text := pred.explainText(nil, nil)
	if text == "" {
		t.Fatal("explain text is empty")
	}
}

func TestEvalBinaryPredicateExplain(t *testing.T) {
	t.Parallel()
	pred := binaryPredicate{
		operator: "&",
		left: comparisonPredicate{
			operator: "=",
			left:     numberExpression{value: 1},
			right:    numberExpression{value: 2},
		},
		right: comparisonPredicate{
			operator: "=",
			left:     numberExpression{value: 3},
			right:    numberExpression{value: 4},
		},
	}
	failed := pred.explain(nil, nil)
	if len(failed) != 2 {
		t.Fatalf("explain failed = %v, want 2 entries", failed)
	}
}
