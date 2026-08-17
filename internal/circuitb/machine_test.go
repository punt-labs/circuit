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

	result, err := machine.AdvanceFromWithBooleans("implement", "red", map[string]bool{"testSuitePassed": false})
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

func TestTDDFlowUsesTestSuitePassedNegation(t *testing.T) {
	t.Parallel()
	machine, err := LoadFile(filepath.Join("..", "..", "machines", "tdd-flow.mch"))
	if err != nil {
		t.Fatalf("load tdd-flow: %v", err)
	}

	if variables := machine.BooleanVariables(); len(variables) != 2 || variables[0] != "codeQualityPassed" || variables[1] != "testSuitePassed" {
		t.Fatalf("BooleanVariables = %v, want [codeQualityPassed testSuitePassed]", variables)
	}

	blocked, err := machine.AdvanceFromWithBooleans("writeTest", "spec", map[string]bool{"testSuitePassed": true})
	if err != nil {
		t.Fatalf("advance writeTest with passing suite: %v", err)
	}
	if blocked.Allowed {
		t.Fatal("writeTest allowed while test suite is passing")
	}

	allowed, err := machine.AdvanceFromWithBooleans("writeTest", "spec", map[string]bool{"testSuitePassed": false})
	if err != nil {
		t.Fatalf("advance writeTest with failing suite: %v", err)
	}
	if !allowed.Allowed || allowed.To != "red" {
		t.Fatalf("writeTest with failing suite = %#v, want red", allowed)
	}
}

func TestPRWatchRequiresChecksAndReviewBeforeFixed(t *testing.T) {
	t.Parallel()
	machine, err := LoadFile(filepath.Join("..", "..", "machines", "pr-watch.mch"))
	if err != nil {
		t.Fatalf("load pr-watch: %v", err)
	}

	if variables := machine.BooleanVariables(); len(variables) != 2 || variables[0] != "checksGreen" || variables[1] != "reviewClean" {
		t.Fatalf("BooleanVariables = %v, want [checksGreen reviewClean]", variables)
	}

	blocked, err := machine.AdvanceFromWithBooleans("fixed", "fixing", map[string]bool{"checksGreen": false, "reviewClean": false})
	if err != nil {
		t.Fatalf("advance fixed with nothing green: %v", err)
	}
	if blocked.Allowed {
		t.Fatal("fixed allowed when checks and review both failing")
	}

	blockedChecks, err := machine.AdvanceFromWithBooleans("fixed", "fixing", map[string]bool{"checksGreen": true, "reviewClean": false})
	if err != nil {
		t.Fatalf("advance fixed with only checks green: %v", err)
	}
	if blockedChecks.Allowed {
		t.Fatal("fixed allowed when review not clean")
	}

	allowed, err := machine.AdvanceFromWithBooleans("fixed", "fixing", map[string]bool{"checksGreen": true, "reviewClean": true})
	if err != nil {
		t.Fatalf("advance fixed with both green: %v", err)
	}
	if !allowed.Allowed || allowed.To != "done" {
		t.Fatalf("fixed = %#v, want done", allowed)
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

func TestTDDFlowQualityReviewDrivesRefactoring(t *testing.T) {
	t.Parallel()
	machine, err := LoadFile(filepath.Join("..", "..", "machines", "tdd-flow.mch"))
	if err != nil {
		t.Fatalf("load tdd-flow: %v", err)
	}

	if variables := machine.BooleanVariables(); len(variables) != 2 || variables[0] != "codeQualityPassed" || variables[1] != "testSuitePassed" {
		t.Fatalf("BooleanVariables = %v, want [codeQualityPassed testSuitePassed]", variables)
	}

	review, err := machine.AdvanceFromWithBooleans("reviewQuality", "green", map[string]bool{"testSuitePassed": true, "codeQualityPassed": false})
	if err != nil {
		t.Fatalf("advance reviewQuality: %v", err)
	}
	if !review.Allowed || review.To != "qualityReview" {
		t.Fatalf("reviewQuality = %#v, want qualityReview", review)
	}

	blockedFinish, err := machine.AdvanceFromWithBooleans("finish", "qualityReview", map[string]bool{"testSuitePassed": true, "codeQualityPassed": false})
	if err != nil {
		t.Fatalf("advance finish with failed quality: %v", err)
	}
	if blockedFinish.Allowed {
		t.Fatal("finish allowed when code quality failed")
	}

	refactor, err := machine.AdvanceFromWithBooleans("refactor", "qualityReview", map[string]bool{"testSuitePassed": true, "codeQualityPassed": false})
	if err != nil {
		t.Fatalf("advance refactor with failed quality: %v", err)
	}
	if !refactor.Allowed || refactor.To != "refactoring" {
		t.Fatalf("refactor = %#v, want refactoring", refactor)
	}

	finished, err := machine.AdvanceFromWithBooleans("finish", "qualityReview", map[string]bool{"testSuitePassed": true, "codeQualityPassed": true})
	if err != nil {
		t.Fatalf("advance finish with passed quality: %v", err)
	}
	if !finished.Allowed || finished.To != "done" {
		t.Fatalf("finish = %#v, want done", finished)
	}
}

func TestTDDFlowRequiresQualityReviewBeforeFinish(t *testing.T) {
	t.Parallel()
	machine, err := LoadFile(filepath.Join("..", "..", "machines", "tdd-flow.mch"))
	if err != nil {
		t.Fatalf("load tdd-flow: %v", err)
	}

	blocked, err := machine.AdvanceFromWithBooleans("finish", "green", map[string]bool{"testSuitePassed": true, "codeQualityPassed": true})
	if err != nil {
		t.Fatalf("advance finish from green: %v", err)
	}
	if blocked.Allowed {
		t.Fatal("finish allowed from green before quality review")
	}

	reviewed, err := machine.AdvanceFromWithBooleans("reviewQuality", "green", map[string]bool{"testSuitePassed": true, "codeQualityPassed": true})
	if err != nil {
		t.Fatalf("advance reviewQuality from green: %v", err)
	}
	if !reviewed.Allowed || reviewed.To != "qualityReview" {
		t.Fatalf("reviewQuality from green = %#v, want qualityReview", reviewed)
	}

	finished, err := machine.AdvanceFromWithBooleans("finish", "qualityReview", map[string]bool{"testSuitePassed": true, "codeQualityPassed": true})
	if err != nil {
		t.Fatalf("advance finish from qualityReview: %v", err)
	}
	if !finished.Allowed || finished.To != "done" {
		t.Fatalf("finish from qualityReview = %#v, want done", finished)
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
