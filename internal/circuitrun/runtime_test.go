package circuitrun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResumeRejectsMalformedSuspendedRuntime(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	path := filepath.Join(root, ".tmp", "circuit.suspended.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create suspended dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("write malformed suspended runtime: %v", err)
	}

	_, err := Resume(root)

	if err == nil {
		t.Fatal("resume malformed runtime returned nil error")
	}
}

func TestRuntimeWithoutRunHasNoopSuspendAndErrorsOnStatusAdvance(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume empty runtime: %v", err)
	}
	if err := runtime.Suspend(); err != nil {
		t.Fatalf("suspend empty runtime: %v", err)
	}
	if _, err := runtime.Status(); err == nil {
		t.Fatal("status without run returned nil error")
	}
	if _, err := runtime.Advance("start"); err == nil {
		t.Fatal("advance without run returned nil error")
	}
}

func TestRuntimeStartMissingMachineFails(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume empty runtime: %v", err)
	}
	if _, _, err := runtime.Start("missing"); err == nil {
		t.Fatal("start missing machine returned nil error")
	}
}

func TestSessionLifecycleStates(t *testing.T) {
	t.Parallel()
	root := testRoot(t)

	// unloaded: no session
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if runtime.IsActive() {
		t.Fatal("new runtime should not be active")
	}
	if runtime.SessionState() != SessionUnloaded {
		t.Fatalf("new runtime session state = %s, want unloaded", runtime.SessionState())
	}

	// start: unloaded -> active
	if _, _, err := runtime.Start("build-job"); err != nil {
		t.Fatalf("start: %v", err)
	}
	if !runtime.IsActive() {
		t.Fatal("started runtime should be active")
	}
	if runtime.SessionState() != SessionActive {
		t.Fatalf("started session state = %s, want active", runtime.SessionState())
	}

	// suspend + resume: active -> suspended -> active
	if err := runtime.Suspend(); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	resumed, err := Resume(root)
	if err != nil {
		t.Fatalf("resume after suspend: %v", err)
	}
	if !resumed.IsActive() {
		t.Fatal("resumed runtime should be active")
	}
	if resumed.SessionState() != SessionActive {
		t.Fatalf("resumed session state = %s, want active", resumed.SessionState())
	}

	// stop: active -> stopped
	if err := resumed.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if resumed.IsActive() {
		t.Fatal("stopped runtime should not be active")
	}
	if resumed.SessionState() != SessionStopped {
		t.Fatalf("stopped session state = %s, want stopped", resumed.SessionState())
	}
	if err := resumed.Stop(); err != nil {
		t.Fatalf("idempotent stop: %v", err)
	}
}

func TestAutoStopOnTerminalState(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if _, _, err := runtime.Start("build-job"); err != nil {
		t.Fatalf("start: %v", err)
	}

	// advance to running
	if _, err := runtime.Advance("start"); err != nil {
		t.Fatalf("advance start: %v", err)
	}
	if !runtime.IsActive() {
		t.Fatal("should still be active after non-terminal advance")
	}

	// advance to done (terminal)
	report, err := runtime.Advance("finish")
	if err != nil {
		t.Fatalf("advance finish: %v", err)
	}
	if !report.Allowed || report.To != "done" {
		t.Fatalf("advance finish = %#v, want allowed to done", report)
	}
	if runtime.IsActive() {
		t.Fatal("should not be active after terminal advance")
	}
	if runtime.SessionState() != SessionStopped {
		t.Fatalf("terminal session state = %s, want stopped", runtime.SessionState())
	}

	// suspend after auto-stop should clear the file
	if err := runtime.Suspend(); err != nil {
		t.Fatalf("suspend after auto-stop: %v", err)
	}
	restoredRuntime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume after auto-stop: %v", err)
	}
	if restoredRuntime.IsActive() {
		t.Fatal("resumed after auto-stop should not be active")
	}
}

func TestStatusWorksAndAdvanceFailsAfterAutoStop(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if _, _, err := runtime.Start("build-job"); err != nil {
		t.Fatalf("start: %v", err)
	}
	if _, err := runtime.Advance("start"); err != nil {
		t.Fatalf("advance start: %v", err)
	}
	if _, err := runtime.Advance("finish"); err != nil {
		t.Fatalf("advance finish: %v", err)
	}

	// Session auto-stopped. Status can inspect it, but Advance should fail.
	status, err := runtime.Status()
	if err != nil {
		t.Fatalf("status after auto-stop: %v", err)
	}
	if status.SessionState != SessionStopped {
		t.Fatalf("status session state = %s, want stopped", status.SessionState)
	}
	if _, err := runtime.Advance("start"); err == nil {
		t.Fatal("advance after auto-stop returned nil error")
	}
}

func TestStartWithFullPathMachine(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	fullPath := filepath.Join(root, "machines", "build-job.mch")
	_, status, err := runtime.Start(fullPath)
	if err != nil {
		t.Fatalf("start full path: %v", err)
	}
	if status.Current != "idle" {
		t.Fatalf("start full path current = %s, want idle", status.Current)
	}
}

func TestRuntimeSuspendsAndResumes(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume empty runtime: %v", err)
	}
	if _, _, err := runtime.Start("build-job"); err != nil {
		t.Fatalf("start build-job: %v", err)
	}
	if _, err := runtime.Advance("start"); err != nil {
		t.Fatalf("advance start: %v", err)
	}
	if err := runtime.Suspend(); err != nil {
		t.Fatalf("suspend: %v", err)
	}

	resumed, err := Resume(root)
	if err != nil {
		t.Fatalf("resume suspended runtime: %v", err)
	}
	status, err := resumed.Status()
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.MachineName != "build-job" || status.Current != "running" {
		t.Fatalf("status = %s/%s, want build-job/running", status.MachineName, status.Current)
	}
}

func TestRuntimeListsMachinesAndReportsSuspendedPath(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume empty runtime: %v", err)
	}
	machines, err := runtime.ListMachines()
	if err != nil {
		t.Fatalf("list machines: %v", err)
	}
	if len(machines) != 4 || machines[0] != "build-job" || machines[1] != "retry-flow" || machines[2] != "review-flow" || machines[3] != "tdd-flow" {
		t.Fatalf("machines = %v, want build-job/retry-flow/review-flow/tdd-flow", machines)
	}
	if runtime.SuspendedPath() != filepath.Join(root, ".tmp", "circuit.suspended.json") {
		t.Fatalf("suspended path = %s", runtime.SuspendedPath())
	}
}

func TestRuntimeStopPreservesStoppedSession(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume empty runtime: %v", err)
	}
	if _, _, err := runtime.Start("build-job"); err != nil {
		t.Fatalf("start build-job: %v", err)
	}
	if err := runtime.Suspend(); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	if err := runtime.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	resumed, err := Resume(root)
	if err != nil {
		t.Fatalf("resume after stop: %v", err)
	}
	status, err := resumed.Status()
	if err != nil {
		t.Fatalf("status after stop: %v", err)
	}
	if status.SessionState != SessionStopped {
		t.Fatalf("status session state = %s, want stopped", status.SessionState)
	}
}

func TestRuntimeRunsBoundCheck(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	writeRegistry(t, root, "true")
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume empty runtime: %v", err)
	}
	if _, _, err := runtime.Start("review-flow"); err != nil {
		t.Fatalf("start review-flow: %v", err)
	}
	report, err := runtime.Advance("requestReview")
	if err != nil {
		t.Fatalf("advance requestReview: %v", err)
	}
	if !report.Allowed || report.To != "codeReview" {
		t.Fatalf("advance report = %#v, want allowed to codeReview", report)
	}
	check := report.Checks["makeCheckPassed"]
	if !check.LastResult || check.Invocations != 1 {
		t.Fatalf("check runtime = %#v, want true once", check)
	}
}

func TestRuntimeRejectsUnknownCheckRegistryEntry(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	path := filepath.Join(root, "machines", "review-flow.checks.yaml")
	if err := os.WriteFile(path, []byte("checks:\n  makeCheckPassed:\n    use: missing\n"), 0o600); err != nil {
		t.Fatalf("write bindings: %v", err)
	}
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume empty runtime: %v", err)
	}
	_, _, err = runtime.Start("review-flow")
	if err == nil || !strings.Contains(err.Error(), "unknown registry") {
		t.Fatalf("start error = %v, want unknown registry", err)
	}
}

func TestRuntimeRejectsNonBooleanCheck(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	content := []byte("checks:\n  makeCheck:\n    kind: command\n    command: true\n    returns: NAT\n")
	if err := os.WriteFile(filepath.Join(root, "machines", "check-registry.yaml"), content, 0o600); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume empty runtime: %v", err)
	}
	_, _, err = runtime.Start("review-flow")
	if err == nil || !strings.Contains(err.Error(), "returning BOOL") {
		t.Fatalf("start error = %v, want BOOL registry error", err)
	}
}

func TestRuntimeBlocksFailedBoundCheck(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	writeRegistry(t, root, "false")
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume empty runtime: %v", err)
	}
	if _, _, err := runtime.Start("review-flow"); err != nil {
		t.Fatalf("start review-flow: %v", err)
	}
	report, err := runtime.Advance("requestReview")
	if err != nil {
		t.Fatalf("advance requestReview: %v", err)
	}
	if report.Allowed {
		t.Fatalf("advance unexpectedly allowed: %#v", report)
	}
	check := report.Checks["makeCheckPassed"]
	if check.LastResult || check.Invocations != 1 {
		t.Fatalf("check runtime = %#v, want false once", check)
	}
}

func TestRuntimeRetryAfterBlockedCheck(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	writeAlternatingRegistry(t, root)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if _, _, err := runtime.Start("retry-flow"); err != nil {
		t.Fatalf("start retry-flow: %v", err)
	}

	// First advance: alternating check fails (invocation 1 = odd)
	report1, err := runtime.Advance("proceed")
	if err != nil {
		t.Fatalf("advance 1: %v", err)
	}
	if report1.Allowed {
		t.Fatal("advance 1 should be blocked")
	}
	check1 := report1.Checks["gateOpen"]
	if check1.LastResult || check1.Invocations != 1 {
		t.Fatalf("check 1 = %#v, want false/1", check1)
	}

	// Second advance: alternating check passes (invocation 2 = even)
	report2, err := runtime.Advance("proceed")
	if err != nil {
		t.Fatalf("advance 2: %v", err)
	}
	if !report2.Allowed || report2.To != "done" {
		t.Fatalf("advance 2 = %#v, want allowed to done", report2)
	}
	check2 := report2.Checks["gateOpen"]
	if !check2.LastResult || check2.Invocations != 2 {
		t.Fatalf("check 2 = %#v, want true/2", check2)
	}
}

func TestRuntimeBlocksTDDFlowWithoutFailingTest(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	writeTDDRegistry(t, root, "false", "true")
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if _, _, err := runtime.Start("tdd-flow"); err != nil {
		t.Fatalf("start tdd-flow: %v", err)
	}

	report, err := runtime.Advance("writeTest")
	if err != nil {
		t.Fatalf("advance writeTest: %v", err)
	}
	if report.Allowed {
		t.Fatalf("writeTest unexpectedly allowed: %#v", report)
	}
	check := report.Checks["failingTestObserved"]
	if check.LastResult || check.Invocations != 1 {
		t.Fatalf("failingTestObserved check = %#v, want false once", check)
	}
	if !runtime.IsActive() {
		t.Fatal("externally gated tdd-flow should remain active while blocked")
	}
}

func TestRuntimeAdvancesTDDFlowHappyPath(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	writeTDDRegistry(t, root, "true", "true")
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if _, _, err := runtime.Start("tdd-flow"); err != nil {
		t.Fatalf("start tdd-flow: %v", err)
	}

	for _, step := range []struct {
		event string
		to    string
	}{
		{event: "writeTest", to: "red"},
		{event: "implement", to: "green"},
		{event: "refactor", to: "refactoring"},
		{event: "keepGreen", to: "green"},
		{event: "finish", to: "done"},
	} {
		report, advanceErr := runtime.Advance(step.event)
		if advanceErr != nil {
			t.Fatalf("advance %s: %v", step.event, advanceErr)
		}
		if !report.Allowed || report.To != step.to {
			t.Fatalf("advance %s = %#v, want %s", step.event, report, step.to)
		}
	}
	if runtime.IsActive() {
		t.Fatal("tdd-flow should auto-stop at done")
	}
}

func testRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	machines := filepath.Join(root, "machines")
	if err := os.MkdirAll(machines, 0o700); err != nil {
		t.Fatalf("create machines dir: %v", err)
	}
	for _, name := range []string{"build-job.mch", "review-flow.mch", "review-flow.checks.yaml", "retry-flow.mch", "retry-flow.checks.yaml", "tdd-flow.mch", "tdd-flow.checks.yaml", "check-registry.yaml", "alternating-check.sh"} {
		copyFixture(t, filepath.Join("..", "..", "machines", name), filepath.Join(machines, name))
	}
	return root
}

func writeAlternatingRegistry(t *testing.T, root string) {
	t.Helper()
	script := filepath.Join(root, "machines", "alternating-check.sh")
	statePath := filepath.Join(root, ".tmp", "alternating-check.state")
	content := []byte("checks:\n  alternatingCheck:\n    kind: command\n    command: sh " + script + " " + statePath + "\n    returns: BOOL\n")
	path := filepath.Join(root, "machines", "check-registry.yaml")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write alternating registry: %v", err)
	}
}

func writeRegistry(t *testing.T, root string, command string) {
	t.Helper()
	if strings.TrimSpace(command) == "" {
		t.Fatal("command must not be empty")
	}
	content := []byte("checks:\n  makeCheck:\n    kind: command\n    command: " + command + "\n    returns: BOOL\n")
	path := filepath.Join(root, "machines", "check-registry.yaml")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write registry: %v", err)
	}
}

func writeTDDRegistry(t *testing.T, root string, failingTestCommand string, testSuiteCommand string) {
	t.Helper()
	if strings.TrimSpace(failingTestCommand) == "" || strings.TrimSpace(testSuiteCommand) == "" {
		t.Fatal("commands must not be empty")
	}
	content := []byte("checks:\n  failingTestObserved:\n    kind: command\n    command: " + failingTestCommand + "\n    returns: BOOL\n  testSuitePassed:\n    kind: command\n    command: " + testSuiteCommand + "\n    returns: BOOL\n")
	path := filepath.Join(root, "machines", "check-registry.yaml")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write tdd registry: %v", err)
	}
}

func copyFixture(t *testing.T, source string, destination string) {
	t.Helper()
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read fixture %s: %v", source, err)
	}
	if err := os.WriteFile(destination, content, 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", destination, err)
	}
}
