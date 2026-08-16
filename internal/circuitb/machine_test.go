package circuitb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBuildJobState(t *testing.T) {
	t.Parallel()
	machine := loadFixture(t)

	report, err := machine.State(nil)

	if err != nil {
		t.Fatalf("state failed: %v", err)
	}
	if report.Current != "idle" {
		t.Fatalf("current state = %q, want idle", report.Current)
	}
	if !containsCall(report.Enabled, "Advance(start)") {
		t.Fatalf("enabled calls = %v, want Advance(start)", report.Enabled)
	}
	if !containsCall(report.Blocked, "Advance(finish)") {
		t.Fatalf("blocked calls = %v, want Advance(finish)", report.Blocked)
	}
}

func TestAdvanceStart(t *testing.T) {
	t.Parallel()
	machine := loadFixture(t)

	result, err := machine.Advance("start", nil)

	if err != nil {
		t.Fatalf("advance start failed: %v", err)
	}
	if !result.Allowed {
		t.Fatalf("advance start blocked: %v", result.Failed)
	}
	if result.From != "idle" || result.To != "running" {
		t.Fatalf("advance start = %s -> %s, want idle -> running", result.From, result.To)
	}
}

func TestAdvanceFinishBlockedInitially(t *testing.T) {
	t.Parallel()
	machine := loadFixture(t)

	result, err := machine.Advance("finish", nil)

	if err != nil {
		t.Fatalf("advance finish returned unexpected error: %v", err)
	}
	if result.Allowed {
		t.Fatalf("advance finish allowed from initial state")
	}
	if result.From != "idle" || result.To != "" {
		t.Fatalf("blocked finish = %s -> %s, want idle -> empty", result.From, result.To)
	}
}

func TestBooleanVariables(t *testing.T) {
	t.Parallel()
	machine, err := LoadFile(filepath.Join("..", "..", "machines", "review-flow.mch"))
	if err != nil {
		t.Fatalf("load review-flow: %v", err)
	}

	variables := machine.BooleanVariables()

	if len(variables) != 1 || variables[0] != "makeCheckPassed" {
		t.Fatalf("BooleanVariables = %v, want makeCheckPassed", variables)
	}
}

func TestBuildJobHasNoBooleanVariables(t *testing.T) {
	t.Parallel()
	machine := loadFixture(t)

	variables := machine.BooleanVariables()

	if len(variables) != 0 {
		t.Fatalf("BooleanVariables = %v, want empty", variables)
	}
}

func TestReviewFlowBooleanPrecondition(t *testing.T) {
	t.Parallel()
	machine, err := LoadFile(filepath.Join("..", "..", "machines", "review-flow.mch"))
	if err != nil {
		t.Fatalf("load review-flow: %v", err)
	}

	blocked, err := machine.AdvanceFromWithBooleans("requestReview", "coding", map[string]bool{"makeCheckPassed": false})
	if err != nil {
		t.Fatalf("advance with false check: %v", err)
	}
	if blocked.Allowed {
		t.Fatalf("requestReview allowed with makeCheckPassed false")
	}

	allowed, err := machine.AdvanceFromWithBooleans("requestReview", "coding", map[string]bool{"makeCheckPassed": true})
	if err != nil {
		t.Fatalf("advance with true check: %v", err)
	}
	if !allowed.Allowed || allowed.To != "codeReview" {
		t.Fatalf("advance with true check = %#v, want codeReview", allowed)
	}
}

func TestReviewFlowStateUsesBooleanFacts(t *testing.T) {
	t.Parallel()
	machine, err := LoadFile(filepath.Join("..", "..", "machines", "review-flow.mch"))
	if err != nil {
		t.Fatalf("load review-flow: %v", err)
	}

	report, err := machine.StateAtWithBooleans("coding", map[string]bool{"makeCheckPassed": true})
	if err != nil {
		t.Fatalf("state with boolean facts: %v", err)
	}
	if !containsCall(report.Enabled, "Advance(requestReview)") {
		t.Fatalf("enabled calls = %v, want Advance(requestReview)", report.Enabled)
	}
}

func TestTDDFlowImplementBlockedWithoutSuite(t *testing.T) {
	t.Parallel()
	machine, err := LoadFile(filepath.Join("..", "..", "machines", "tdd-flow.mch"))
	if err != nil {
		t.Fatalf("load tdd-flow: %v", err)
	}

	// implement requires testSuitePassed = TRUE
	// This test will fail until we verify the blocking diagnostic message
	// includes the variable name.
	result, err := machine.AdvanceFromWithBooleans("implement", "red", map[string]bool{"failingTestObserved": true, "testSuitePassed": false})
	if err != nil {
		t.Fatalf("advance implement: %v", err)
	}
	if result.Allowed {
		t.Fatal("implement allowed without passing test suite")
	}
	if len(result.Failed) == 0 || !strings.Contains(result.Failed[0], "testSuitePassed") {
		t.Fatalf("blocked diagnostics = %v, want testSuitePassed mention", result.Failed)
	}
}

func TestNotPredicateNegatesInnerResult(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "NegateFlag.mch")
	source := `MACHINE NegateFlag
SETS
    STATE = {ready, done};
    TRANSITION = {go}
VARIABLES
    current,
    flag
INVARIANT
    current : STATE &
    flag : BOOL
INITIALISATION
    current := ready ||
    flag := FALSE
OPERATIONS
    Advance(evt) =
        PRE
            evt : TRANSITION &
            current = ready &
            evt = go &
            not(flag = TRUE)
        THEN
            current := done
        END
END
`
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatalf("write negation fixture: %v", err)
	}
	machine, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load negation fixture: %v", err)
	}

	blocked, err := machine.AdvanceFromWithBooleans("go", "ready", map[string]bool{"flag": true})
	if err != nil {
		t.Fatalf("advance go with flag true: %v", err)
	}
	if blocked.Allowed {
		t.Fatal("go allowed with flag true; expected blocked")
	}

	allowed, err := machine.AdvanceFromWithBooleans("go", "ready", map[string]bool{"flag": false})
	if err != nil {
		t.Fatalf("advance go with flag false: %v", err)
	}
	if !allowed.Allowed || allowed.To != "done" {
		t.Fatalf("go with flag false = %#v, want done", allowed)
	}
}

func TestTDDFlowBooleanPreconditions(t *testing.T) {
	t.Parallel()
	machine, err := LoadFile(filepath.Join("..", "..", "machines", "tdd-flow.mch"))
	if err != nil {
		t.Fatalf("load tdd-flow: %v", err)
	}

	blocked, err := machine.AdvanceFromWithBooleans("writeTest", "spec", map[string]bool{"failingTestObserved": false, "testSuitePassed": false})
	if err != nil {
		t.Fatalf("advance writeTest with false failing test: %v", err)
	}
	if blocked.Allowed {
		t.Fatal("writeTest allowed before failing test observed")
	}

	allowed, err := machine.AdvanceFromWithBooleans("writeTest", "spec", map[string]bool{"failingTestObserved": true, "testSuitePassed": false})
	if err != nil {
		t.Fatalf("advance writeTest with true failing test: %v", err)
	}
	if !allowed.Allowed || allowed.To != "red" {
		t.Fatalf("writeTest with failing test = %#v, want red", allowed)
	}

	blockedFinish, err := machine.AdvanceFromWithBooleans("finish", "green", map[string]bool{"failingTestObserved": true, "testSuitePassed": false})
	if err != nil {
		t.Fatalf("advance finish with false suite: %v", err)
	}
	if blockedFinish.Allowed {
		t.Fatal("finish allowed before test suite passed")
	}
}

func TestLoadRejectsUnsupportedAny(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "Unsupported.mch")
	content := strings.ReplaceAll(simpleMachineSource(t), "current := idle", "ANY xx WHERE xx : STATE THEN current := xx END")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write unsupported fixture: %v", err)
	}

	_, err := LoadFile(path)

	if err == nil {
		t.Fatal("unsupported ANY machine loaded without error")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unsupported error = %q", err.Error())
	}
}

func loadFixture(t *testing.T) Machine {
	t.Helper()
	machine, err := LoadFile(filepath.Join("..", "..", "machines", "build-job.mch"))
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	return machine
}

func simpleMachineSource(t *testing.T) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "machines", "build-job.mch"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(content)
}

func containsCall(calls []CallStatus, name string) bool {
	for _, call := range calls {
		if call.Call == name {
			return true
		}
	}
	return false
}
