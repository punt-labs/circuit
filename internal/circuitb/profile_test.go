package circuitb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProfileSupportsNumbersComparisonsAndLiteralSets(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "Numbers.mch")
	content := `MACHINE Numbers
SETS
    STATE = {idle, done};
    TRANSITION = {finish}
VARIABLES
    current,
    count,
    flag
INVARIANT
    current : STATE &
    count : NAT &
    flag : BOOL
INITIALISATION
    current := idle ||
    count := 1 ||
    flag := TRUE
OPERATIONS
    Advance(evt) =
        PRE
            evt : TRANSITION &
            current = idle &
            count >= 1 &
            flag : {TRUE}
        THEN
            IF current = idle & evt = finish THEN
                current := done
            ELSE
                current := idle
            END
        END
END
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write profile machine: %v", err)
	}

	machine, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load profile machine: %v", err)
	}
	result, err := machine.Advance("finish", nil)
	if err != nil {
		t.Fatalf("advance finish: %v", err)
	}
	if !result.Allowed || result.To != "done" {
		t.Fatalf("advance = %#v, want allowed to done", result)
	}
}

func TestProfileReportsFailedLiteralSetMembership(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "Membership.mch")
	content := `MACHINE Membership
SETS
    STATE = {idle, done};
    TRANSITION = {finish}
VARIABLES
    current,
    flag
INVARIANT
    current : STATE &
    flag : BOOL
INITIALISATION
    current := idle ||
    flag := TRUE
OPERATIONS
    Advance(evt) =
        PRE
            evt : TRANSITION &
            current = idle &
            flag : {TRUE}
        THEN
            current := done
        END
END
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write membership machine: %v", err)
	}
	machine, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load membership machine: %v", err)
	}

	result, err := machine.AdvanceFromWithBooleans("finish", "idle", map[string]bool{"flag": false})

	if err != nil {
		t.Fatalf("advance with false flag: %v", err)
	}
	if result.Allowed {
		t.Fatal("advance allowed with failed membership")
	}
	if len(result.Failed) == 0 || !strings.Contains(result.Failed[0], "flag : set literal") {
		t.Fatalf("failed preconditions = %v, want flag membership", result.Failed)
	}
}

func TestProfileSupportsParallelAssignmentOperation(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "Parallel.mch")
	content := `MACHINE Parallel
SETS
    STATE = {idle, done};
    TRANSITION = {finish}
VARIABLES
    current,
    count
INVARIANT
    current : STATE &
    count : NAT
INITIALISATION
    current := idle ||
    count := 0
OPERATIONS
    Advance(evt) =
        PRE
            evt : TRANSITION &
            current = idle
        THEN
            current := done ||
            count := 1
        END
END
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write parallel machine: %v", err)
	}
	machine, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load parallel machine: %v", err)
	}

	result, err := machine.Advance("finish", nil)

	if err != nil {
		t.Fatalf("advance parallel machine: %v", err)
	}
	if !result.Allowed || result.To != "done" {
		t.Fatalf("advance = %#v, want allowed to done", result)
	}
}

func TestLoadRejectsMissingCurrentInvariant(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "MissingCurrent.mch")
	content := `MACHINE MissingCurrent
SETS
    STATE = {idle, done};
    TRANSITION = {finish}
VARIABLES
    mode
INVARIANT
    mode : STATE
INITIALISATION
    mode := idle
OPERATIONS
    Advance(evt) =
        PRE
            evt : TRANSITION
        THEN
            mode := done
        END
END
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write missing-current machine: %v", err)
	}

	_, err := LoadFile(path)

	if err == nil || !strings.Contains(err.Error(), "current variable") {
		t.Fatalf("missing current error = %v", err)
	}
}

func TestLoadRejectsUnknownInitialisationVariable(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "UnknownInit.mch")
	content := `MACHINE UnknownInit
SETS
    STATE = {idle, done};
    TRANSITION = {finish}
VARIABLES
    current
INVARIANT
    current : STATE
INITIALISATION
    current := idle ||
    ghost := done
OPERATIONS
    Advance(evt) =
        PRE
            evt : TRANSITION
        THEN
            current := done
        END
END
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write unknown-init machine: %v", err)
	}

	_, err := LoadFile(path)

	if err == nil || !strings.Contains(err.Error(), "unknown variable ghost") {
		t.Fatalf("unknown init error = %v", err)
	}
}

func TestProfileReportsFailedNamedSetMembership(t *testing.T) {
	t.Parallel()
	machine := loadFixture(t)

	result, err := machine.Advance("bogus", nil)

	if err != nil {
		t.Fatalf("advance bogus: %v", err)
	}
	if result.Allowed {
		t.Fatal("bogus transition allowed")
	}
	if len(result.Failed) == 0 || !strings.Contains(result.Failed[0], "evt : TRANSITION") {
		t.Fatalf("failed preconditions = %v, want transition membership", result.Failed)
	}
}

func TestProfileReportsFailedNumericComparison(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "NumericBlock.mch")
	content := `MACHINE NumericBlock
SETS
    STATE = {idle, done};
    TRANSITION = {finish}
VARIABLES
    current,
    count
INVARIANT
    current : STATE &
    count : NAT
INITIALISATION
    current := idle ||
    count := 0
OPERATIONS
    Advance(evt) =
        PRE
            evt : TRANSITION &
            current = idle &
            count > 0
        THEN
            current := done
        END
END
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write numeric machine: %v", err)
	}
	machine, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load numeric machine: %v", err)
	}

	result, err := machine.Advance("finish", nil)

	if err != nil {
		t.Fatalf("advance numeric machine: %v", err)
	}
	if result.Allowed {
		t.Fatal("numeric transition allowed")
	}
	if len(result.Failed) == 0 || !strings.Contains(result.Failed[0], "count > 0") {
		t.Fatalf("failed preconditions = %v, want count comparison", result.Failed)
	}
}

func TestProfileSupportsIfElseFallback(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "Fallback.mch")
	content := `MACHINE Fallback
SETS
    STATE = {idle, running, done};
    TRANSITION = {finish}
VARIABLES
    current
INVARIANT
    current : STATE
INITIALISATION
    current := idle
OPERATIONS
    Advance(evt) =
        PRE
            evt : TRANSITION
        THEN
            IF current = running THEN
                current := done
            ELSE
                current := idle
            END
        END
END
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fallback machine: %v", err)
	}
	machine, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load fallback machine: %v", err)
	}

	result, err := machine.Advance("finish", nil)

	if err != nil {
		t.Fatalf("advance fallback machine: %v", err)
	}
	if !result.Allowed || result.To != "idle" {
		t.Fatalf("fallback result = %#v, want allowed to idle", result)
	}
}

func TestProfileParsesOutputOperationWithExplicitEnd(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "Query.mch")
	content := `MACHINE Query
SETS
    STATE = {idle};
    TRANSITION = {noop}
VARIABLES
    current
INVARIANT
    current : STATE
INITIALISATION
    current := idle
OPERATIONS
    cc <-- Current =
        cc := current
    END;

    Advance(evt) =
        PRE
            evt : TRANSITION
        THEN
            current := idle
        END
END
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write query machine: %v", err)
	}

	if _, err := LoadFile(path); err != nil {
		t.Fatalf("load query machine: %v", err)
	}
}

func TestParserUsesTokenNamesInPredicateError(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "BadPredicate.mch")
	content := `MACHINE BadPredicate
SETS
    STATE = {idle};
    TRANSITION = {noop}
VARIABLES
    current
INVARIANT
    current : STATE
INITIALISATION
    current := idle
OPERATIONS
    Advance(evt) =
        PRE
            evt : TRANSITION current = idle
        THEN
            current := idle
        END
END
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write bad predicate machine: %v", err)
	}

	_, err := LoadFile(path)

	if err == nil || !strings.Contains(err.Error(), "THEN") {
		t.Fatalf("bad predicate error = %v, want THEN token name", err)
	}
}

func TestLoadFileMissingPathFails(t *testing.T) {
	t.Parallel()
	_, err := LoadFile(filepath.Join(t.TempDir(), "missing.mch"))

	if err == nil {
		t.Fatal("missing file loaded without error")
	}
}

func TestRawNodeSpans(t *testing.T) {
	t.Parallel()
	span := Span{Line: 1, Column: 1}
	number := rawNumberExpression{Value: 1, Span: span}
	literalSet := rawLiteralSetExpression{Values: []rawExpression{number}, Span: span}
	unsupported := rawUnsupportedSubstitution{Construct: "ANY", Span: span}
	assignment := rawAssignment{Name: "x", Value: number, Span: span}
	parallel := rawParallelAssignment{Assignments: []rawAssignment{assignment}, Span: span}
	branch := rawGuardedSubstitution{Condition: rawComparisonPredicate{Operator: "=", Left: number, Right: number, Span: span}, Body: assignment}
	conditional := rawIfSubstitution{Branches: []rawGuardedSubstitution{branch}, Span: span}

	if number.expressionSpan() != span || literalSet.setExpressionSpan() != span {
		t.Fatal("expression spans were not preserved")
	}
	if unsupported.substitutionSpan() != span || assignment.substitutionSpan() != span || parallel.substitutionSpan() != span || conditional.substitutionSpan() != span {
		t.Fatal("substitution spans were not preserved")
	}
}

func TestDiagnosticsIncludeHint(t *testing.T) {
	t.Parallel()
	diagnostics := Diagnostics{{Span: Span{File: "machine.mch", Line: 1, Column: 2}, Message: "bad", Hint: "fix it"}}

	text := diagnostics.Error()

	if !strings.Contains(text, "machine.mch:1:2: bad") || !strings.Contains(text, "hint: fix it") {
		t.Fatalf("diagnostic text = %q", text)
	}
}

func TestLexerRejectsUnexpectedCharacter(t *testing.T) {
	t.Parallel()
	_, err := lex("bad.mch", "@")

	if err == nil {
		t.Fatal("lex accepted unexpected character")
	}
}

func TestParserRejectsMissingMachineKeyword(t *testing.T) {
	t.Parallel()
	tokens, err := lex("bad.mch", "BuildJob")
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	_, err = parse(tokens)

	if err == nil {
		t.Fatal("parse accepted missing MACHINE keyword")
	}
}
