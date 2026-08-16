package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBMachineList(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)

	err := cmd.run([]string{"list"})

	if err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "build-job") || !strings.Contains(output, "pr-watch") || !strings.Contains(output, "tdd-flow") {
		t.Fatalf("list output mismatch: %q", output)
	}
}

func TestBMachineLoadValidatesChecks(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)

	if err := cmd.run([]string{"load", "review-flow"}); err != nil {
		t.Fatalf("load returned error: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "loaded: review-flow") || !strings.Contains(output, "makeCheckPassed -> makeCheck: BOOL") {
		t.Fatalf("load output mismatch: %q", output)
	}
}

func TestBMachineScaffoldGeneratesFailingCheckStubs(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)
	if err := os.Remove(filepath.Join(cmd.cwd, "machines", "review-flow.checks.yaml")); err != nil {
		t.Fatalf("remove bindings: %v", err)
	}
	if err := os.Remove(filepath.Join(cmd.cwd, "machines", "check-registry.yaml")); err != nil {
		t.Fatalf("remove registry: %v", err)
	}

	if err := cmd.run([]string{"scaffold", "review-flow"}); err != nil {
		t.Fatalf("scaffold returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "scaffolded: review-flow") {
		t.Fatalf("scaffold output mismatch: %q", stdout.String())
	}
	stdout.Reset()
	if err := cmd.run([]string{"start", "review-flow"}); err != nil {
		t.Fatalf("start with stubs returned error: %v", err)
	}
	stdout.Reset()
	if err := cmd.run([]string{"advance", "requestReview"}); err != nil {
		t.Fatalf("advance with stubs returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "blocked: Advance(requestReview)") {
		t.Fatalf("stub did not block: %q", stdout.String())
	}
}

func TestBMachineStartRejectsUnscaffoldedBooleanMachine(t *testing.T) {
	t.Parallel()
	cmd := testCommand(t, &bytes.Buffer{})
	if err := os.Remove(filepath.Join(cmd.cwd, "machines", "review-flow.checks.yaml")); err != nil {
		t.Fatalf("remove bindings: %v", err)
	}

	err := cmd.run([]string{"start", "review-flow"})

	if err == nil || !strings.Contains(err.Error(), "unbound BOOL variable makeCheckPassed") {
		t.Fatalf("start error = %v, want unbound BOOL variable", err)
	}
}

func TestAdvanceJSONReportsAllowedAndBlockedTransitions(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)
	if err := cmd.run([]string{"start", "build-job"}); err != nil {
		t.Fatalf("start returned error: %v", err)
	}
	stdout.Reset()
	if err := cmd.run([]string{"advance", "--json", "start"}); err != nil {
		t.Fatalf("advance --json start returned error: %v", err)
	}
	var allowed advanceJSONTest
	if err := json.Unmarshal(stdout.Bytes(), &allowed); err != nil {
		t.Fatalf("allowed advance output is not JSON: %v; output %q", err, stdout.String())
	}
	if !allowed.Allowed || allowed.Event != "start" || allowed.From != "idle" || allowed.To != "running" {
		t.Fatalf("allowed advance json = %#v, want idle -> running", allowed)
	}

	stdout.Reset()
	if err := cmd.run([]string{"start", "build-job"}); err != nil {
		t.Fatalf("start second build-job returned error: %v", err)
	}
	sessionID := extractSessionID(t, stdout.String())
	stdout.Reset()
	if err := cmd.run([]string{"advance", "--json", "finish", sessionID}); err != nil {
		t.Fatalf("advance --json finish returned error: %v", err)
	}
	var blocked advanceJSONTest
	if err := json.Unmarshal(stdout.Bytes(), &blocked); err != nil {
		t.Fatalf("blocked advance output is not JSON: %v; output %q", err, stdout.String())
	}
	if blocked.Allowed || blocked.Event != "finish" || blocked.From != "idle" || len(blocked.Failed) == 0 {
		t.Fatalf("blocked advance json = %#v, want blocked finish from idle with failed predicates", blocked)
	}
}

func TestStatusJSONReportsAllSessions(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)

	if err := cmd.run([]string{"start", "build-job"}); err != nil {
		t.Fatalf("start first build-job: %v", err)
	}
	first := extractSessionID(t, stdout.String())
	stdout.Reset()
	if err := cmd.run([]string{"advance", "start", first}); err != nil {
		t.Fatalf("advance first start: %v", err)
	}
	stdout.Reset()
	if err := cmd.run([]string{"start", "build-job"}); err != nil {
		t.Fatalf("start second build-job: %v", err)
	}
	second := extractSessionID(t, stdout.String())
	stdout.Reset()
	if err := cmd.run([]string{"advance", "start", second}); err != nil {
		t.Fatalf("advance second start: %v", err)
	}
	stdout.Reset()
	if err := cmd.run([]string{"advance", "finish", second}); err != nil {
		t.Fatalf("advance second finish: %v", err)
	}

	stdout.Reset()
	if err := cmd.run([]string{"status", "--json"}); err != nil {
		t.Fatalf("status --json returned error: %v", err)
	}
	var reports []statusJSONTest
	if err := json.Unmarshal(stdout.Bytes(), &reports); err != nil {
		t.Fatalf("status --json output is not JSON: %v; output %q", err, stdout.String())
	}
	if len(reports) != 2 {
		t.Fatalf("json report count = %d, want 2: %#v", len(reports), reports)
	}
	assertStatusJSON(t, reports, first, "active", "running")
	assertStatusJSON(t, reports, second, "stopped", "done")
}

func TestBMachineStartStatusAdvance(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)

	if err := cmd.run([]string{"start", "build-job"}); err != nil {
		t.Fatalf("start returned error: %v", err)
	}
	startOutput := stdout.String()
	if !strings.Contains(startOutput, "started: build-job") {
		t.Fatalf("start missing machine name: %q", startOutput)
	}
	if !strings.Contains(startOutput, "current: idle") {
		t.Fatalf("start missing current state: %q", startOutput)
	}

	stdout.Reset()
	if err := cmd.run([]string{"advance", "start"}); err != nil {
		t.Fatalf("advance returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "advanced: idle -> running") {
		t.Fatalf("advance output mismatch: %q", stdout.String())
	}

	stdout.Reset()
	if err := cmd.run([]string{"status"}); err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	statusOutput := stdout.String()
	if !strings.Contains(statusOutput, "machine: build-job") {
		t.Fatalf("status missing machine name: %q", statusOutput)
	}
	if !strings.Contains(statusOutput, "current: running") {
		t.Fatalf("status missing updated state: %q", statusOutput)
	}
	if !strings.Contains(statusOutput, "Advance(finish)") {
		t.Fatalf("status missing enabled finish: %q", statusOutput)
	}
}

func TestBMachineStatusWithoutSessionIsInformational(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)

	if err := cmd.run([]string{"status"}); err != nil {
		t.Fatalf("status without session returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "no session") {
		t.Fatalf("status without session output mismatch: %q", stdout.String())
	}
}

func TestBMachineAdvanceRequiresActiveCircuit(t *testing.T) {
	t.Parallel()
	cmd := testCommand(t, &bytes.Buffer{})

	err := cmd.run([]string{"advance", "start"})

	if err == nil {
		t.Fatal("advance without active circuit returned nil error")
	}
	if !strings.Contains(err.Error(), "no active session") {
		t.Fatalf("advance without active circuit error mismatch: %v", err)
	}
}

func TestBMachineMultipleSessionsCanBeTargeted(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)

	if err := cmd.run([]string{"start", "build-job"}); err != nil {
		t.Fatalf("start 1 returned error: %v", err)
	}
	first := extractSessionID(t, stdout.String())
	stdout.Reset()
	if err := cmd.run([]string{"start", "build-job"}); err != nil {
		t.Fatalf("start 2 returned error: %v", err)
	}
	second := extractSessionID(t, stdout.String())
	if first == second {
		t.Fatalf("duplicate session id %s", first)
	}

	stdout.Reset()
	if err := cmd.run([]string{"status"}); err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	statusOutput := stdout.String()
	if !strings.Contains(statusOutput, "session: "+first) || !strings.Contains(statusOutput, "session: "+second) {
		t.Fatalf("status did not show both sessions: %q", statusOutput)
	}

	stdout.Reset()
	if err := cmd.run([]string{"advance", "start", first}); err != nil {
		t.Fatalf("advance targeted session returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "advanced: idle -> running") {
		t.Fatalf("targeted advance output mismatch: %q", stdout.String())
	}

	stdout.Reset()
	if err := cmd.run([]string{"status", first}); err != nil {
		t.Fatalf("targeted status returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "current: running") {
		t.Fatalf("targeted status did not preserve advanced state: %q", stdout.String())
	}

	stdout.Reset()
	if err := cmd.run([]string{"stop", second}); err != nil {
		t.Fatalf("targeted stop returned error: %v", err)
	}
	stdout.Reset()
	if err := cmd.run([]string{"status"}); err != nil {
		t.Fatalf("status after targeted stop returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "session: "+second) || !strings.Contains(stdout.String(), "session-state: stopped") {
		t.Fatalf("stopped session not visible as stopped: %q", stdout.String())
	}
}

func TestBMachineMultipleSessionsRejectImplicitAdvanceAndStop(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)

	if err := cmd.run([]string{"start", "build-job"}); err != nil {
		t.Fatalf("start 1 returned error: %v", err)
	}
	stdout.Reset()
	if err := cmd.run([]string{"start", "build-job"}); err != nil {
		t.Fatalf("start 2 returned error: %v", err)
	}

	if err := cmd.run([]string{"advance", "start"}); err == nil || !strings.Contains(err.Error(), "multiple active sessions") {
		t.Fatalf("implicit advance error = %v, want ambiguity", err)
	}
	if err := cmd.run([]string{"stop"}); err == nil || !strings.Contains(err.Error(), "multiple active sessions") {
		t.Fatalf("implicit stop error = %v, want ambiguity", err)
	}
}

func TestBMachineAdvanceBlocked(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)

	if err := cmd.run([]string{"start", "build-job"}); err != nil {
		t.Fatalf("start returned error: %v", err)
	}
	stdout.Reset()

	err := cmd.run([]string{"advance", "finish"})

	if err != nil {
		t.Fatalf("blocked advance returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "blocked: Advance(finish)") {
		t.Fatalf("blocked advance output mismatch: %q", stdout.String())
	}
}

func TestReviewFlowRunsBoundCheckAndAdvances(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)
	writeRegistry(t, cmd.cwd, "true")

	if err := cmd.run([]string{"start", "review-flow"}); err != nil {
		t.Fatalf("start returned error: %v", err)
	}
	stdout.Reset()
	if err := cmd.run([]string{"advance", "requestReview"}); err != nil {
		t.Fatalf("advance returned error: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "advanced: coding -> codeReview") {
		t.Fatalf("advance output mismatch: %q", output)
	}
	stdout.Reset()
	if err := cmd.run([]string{"status"}); err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	statusOutput := stdout.String()
	if !strings.Contains(statusOutput, "makeCheckPassed: TRUE (invocations: 1)") {
		t.Fatalf("status missing check metadata: %q", statusOutput)
	}
}

func TestReviewFlowBlocksWhenBoundCheckFails(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)
	writeRegistry(t, cmd.cwd, "false")

	if err := cmd.run([]string{"start", "review-flow"}); err != nil {
		t.Fatalf("start returned error: %v", err)
	}
	stdout.Reset()
	if err := cmd.run([]string{"advance", "requestReview"}); err != nil {
		t.Fatalf("advance returned error: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "blocked: Advance(requestReview)") {
		t.Fatalf("blocked output mismatch: %q", output)
	}
	if !strings.Contains(output, "makeCheckPassed: FALSE (invocations: 1)") {
		t.Fatalf("blocked output missing check metadata: %q", output)
	}
}

func TestPRWatchAdvancesToDone(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)

	if err := cmd.run([]string{"start", "pr-watch"}); err != nil {
		t.Fatalf("start returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "current: watch") {
		t.Fatalf("start output mismatch: %q", stdout.String())
	}
	stdout.Reset()
	if err := cmd.run([]string{"advance", "needsWork"}); err != nil {
		t.Fatalf("advance needsWork returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "advanced: watch -> fixing") {
		t.Fatalf("advance needsWork output mismatch: %q", stdout.String())
	}
	stdout.Reset()
	if err := cmd.run([]string{"advance", "fixed"}); err != nil {
		t.Fatalf("advance fixed returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "advanced: fixing -> done") {
		t.Fatalf("advance fixed output mismatch: %q", stdout.String())
	}
}

func TestRetryFlowBlocksThenAdvances(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)
	writeAlternatingRegistry(t, cmd.cwd)

	if err := cmd.run([]string{"start", "retry-flow"}); err != nil {
		t.Fatalf("start returned error: %v", err)
	}
	stdout.Reset()

	// First advance: blocked (invocation 1 = odd = fail)
	if err := cmd.run([]string{"advance", "proceed"}); err != nil {
		t.Fatalf("advance 1 returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "blocked: Advance(proceed)") {
		t.Fatalf("advance 1 not blocked: %q", stdout.String())
	}
	stdout.Reset()

	// Second advance: allowed (invocation 2 = even = pass)
	if err := cmd.run([]string{"advance", "proceed"}); err != nil {
		t.Fatalf("advance 2 returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "advanced: waiting -> done") {
		t.Fatalf("advance 2 not allowed: %q", stdout.String())
	}
}

func TestTDDFlowAdvancesThroughSessionScopedSuiteStamp(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)
	writeTDDRegistry(t, cmd.cwd, "test -f .tmp/circuit/$CIRCUIT_SESSION_ID/suite-green.stamp")

	if err := cmd.run([]string{"start", "tdd-flow"}); err != nil {
		t.Fatalf("start returned error: %v", err)
	}
	sessionID := extractSessionID(t, stdout.String())
	stampPath := filepath.Join(cmd.cwd, ".tmp", "circuit", sessionID, "suite-green.stamp")
	if err := os.MkdirAll(filepath.Dir(stampPath), 0o700); err != nil {
		t.Fatalf("create stamp dir: %v", err)
	}

	// With suite currently passing, spec -> red is blocked.
	if err := os.WriteFile(stampPath, []byte("green"), 0o600); err != nil {
		t.Fatalf("write stamp: %v", err)
	}
	stdout.Reset()
	if err := cmd.run([]string{"advance", "writeTest"}); err != nil {
		t.Fatalf("advance writeTest with suite green returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "blocked: Advance(writeTest)") {
		t.Fatalf("advance writeTest with suite green output mismatch: %q", stdout.String())
	}

	for _, step := range []struct {
		event      string
		suiteGreen bool
		want       string
	}{
		{event: "writeTest", suiteGreen: false, want: "advanced: spec -> red"},
		{event: "implement", suiteGreen: true, want: "advanced: red -> green"},
		{event: "refactor", suiteGreen: true, want: "advanced: green -> refactoring"},
		{event: "keepGreen", suiteGreen: true, want: "advanced: refactoring -> green"},
		{event: "finish", suiteGreen: true, want: "advanced: green -> done"},
	} {
		if step.suiteGreen {
			if err := os.WriteFile(stampPath, []byte("green"), 0o600); err != nil {
				t.Fatalf("write stamp: %v", err)
			}
		} else {
			if err := os.Remove(stampPath); err != nil && !os.IsNotExist(err) {
				t.Fatalf("remove stamp: %v", err)
			}
		}
		stdout.Reset()
		if err := cmd.run([]string{"advance", step.event}); err != nil {
			t.Fatalf("advance %s returned error: %v", step.event, err)
		}
		if !strings.Contains(stdout.String(), step.want) {
			t.Fatalf("advance %s output = %q, want %q", step.event, stdout.String(), step.want)
		}
	}
}

func TestBMachineStop(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)

	if err := cmd.run([]string{"start", "build-job"}); err != nil {
		t.Fatalf("start returned error: %v", err)
	}
	sessionID := extractSessionID(t, stdout.String())
	stdout.Reset()
	if err := cmd.run([]string{"stop"}); err != nil {
		t.Fatalf("stop returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "stopped") {
		t.Fatalf("stop output mismatch: %q", stdout.String())
	}
	stdout.Reset()
	if err := cmd.run([]string{"stop", sessionID}); err != nil {
		t.Fatalf("idempotent stop returned error: %v", err)
	}
	if err := cmd.run([]string{"status"}); err != nil {
		t.Fatalf("status after stop returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "session-state: stopped") {
		t.Fatalf("status after stop output mismatch: %q", stdout.String())
	}
}

func TestBMachineUnloadStoppedSession(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)

	if err := cmd.run([]string{"start", "build-job"}); err != nil {
		t.Fatalf("start returned error: %v", err)
	}
	sessionID := extractSessionID(t, stdout.String())
	stdout.Reset()
	if err := cmd.run([]string{"stop", sessionID}); err != nil {
		t.Fatalf("stop returned error: %v", err)
	}
	stdout.Reset()
	if err := cmd.run([]string{"unload", sessionID}); err != nil {
		t.Fatalf("unload returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "unloaded") {
		t.Fatalf("unload output mismatch: %q", stdout.String())
	}
	stdout.Reset()
	if err := cmd.run([]string{"status"}); err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "no session") {
		t.Fatalf("status after unload mismatch: %q", stdout.String())
	}
}

func TestBMachineUnloadRejectsActiveSession(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)

	if err := cmd.run([]string{"start", "build-job"}); err != nil {
		t.Fatalf("start returned error: %v", err)
	}
	sessionID := extractSessionID(t, stdout.String())
	err := cmd.run([]string{"unload", sessionID})
	if err == nil || !strings.Contains(err.Error(), "cannot unload active session") {
		t.Fatalf("unload active error = %v, want active rejection", err)
	}
}

func TestBMachineStopWithoutSessionFails(t *testing.T) {
	t.Parallel()
	cmd := testCommand(t, &bytes.Buffer{})

	err := cmd.run([]string{"stop"})

	if err == nil || !strings.Contains(err.Error(), "no session to stop") {
		t.Fatalf("stop without session error = %v, want no session", err)
	}
}

func TestMissingCommandFails(t *testing.T) {
	t.Parallel()
	stderr := &bytes.Buffer{}
	cmd := command{stdout: &bytes.Buffer{}, stderr: stderr}

	err := cmd.run([]string{})

	if err == nil {
		t.Fatal("empty command returned nil error")
	}
}

func TestHelpPrintsUsage(t *testing.T) {
	t.Parallel()
	stderr := &bytes.Buffer{}
	cmd := command{stdout: &bytes.Buffer{}, stderr: stderr}

	err := cmd.run([]string{"help"})

	if err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("help did not print usage: %q", stderr.String())
	}
}

func TestListRejectsExtraArgs(t *testing.T) {
	t.Parallel()
	cmd := testCommand(t, &bytes.Buffer{})

	err := cmd.run([]string{"list", "extra"})

	if err == nil {
		t.Fatal("list with extra args returned nil error")
	}
}

func TestStartRejectsNoArgs(t *testing.T) {
	t.Parallel()
	cmd := testCommand(t, &bytes.Buffer{})

	err := cmd.run([]string{"start"})

	if err == nil {
		t.Fatal("start with no args returned nil error")
	}
}

func TestStatusRejectsTooManyArgs(t *testing.T) {
	t.Parallel()
	cmd := testCommand(t, &bytes.Buffer{})

	err := cmd.run([]string{"status", "one", "two"})

	if err == nil {
		t.Fatal("status with too many args returned nil error")
	}
}

func TestAdvanceRejectsNoArgs(t *testing.T) {
	t.Parallel()
	cmd := testCommand(t, &bytes.Buffer{})

	err := cmd.run([]string{"advance"})

	if err == nil {
		t.Fatal("advance with no args returned nil error")
	}
}

func TestUnloadRejectsNoArgs(t *testing.T) {
	t.Parallel()
	cmd := testCommand(t, &bytes.Buffer{})

	err := cmd.run([]string{"unload"})

	if err == nil {
		t.Fatal("unload with no args returned nil error")
	}
}

func TestStopRejectsTooManyArgs(t *testing.T) {
	t.Parallel()
	cmd := testCommand(t, &bytes.Buffer{})

	err := cmd.run([]string{"stop", "one", "two"})

	if err == nil {
		t.Fatal("stop with too many args returned nil error")
	}
}

type stubBackend struct {
	responses []string
	prompts   []string
}

func (backend *stubBackend) Prompt(message string) (string, error) {
	backend.prompts = append(backend.prompts, message)
	if len(backend.responses) == 0 {
		return "", nil
	}
	response := backend.responses[0]
	backend.responses = backend.responses[1:]
	return response, nil
}

func TestLoadGuidanceReadsMachinePromptFile(t *testing.T) {
	t.Parallel()
	cmd := testCommand(t, &bytes.Buffer{})
	guidance, err := cmd.loadGuidance("tdd-flow", "Add load --json")
	if err != nil {
		t.Fatalf("load guidance: %v", err)
	}
	if guidance.Goal != "Add load --json" {
		t.Fatalf("goal = %q, want task", guidance.Goal)
	}
	for state, want := range map[string]string{
		"spec":        "Write the failing test",
		"red":         "Implement the smallest code",
		"green":       "finish or request refactor",
		"refactoring": "Refactor while keeping checks green",
	} {
		prompt := guidance.States[state].Prompt
		if !strings.Contains(prompt, want) {
			t.Fatalf("%s prompt = %q, want %q", state, prompt, want)
		}
	}
	if guidance.States["spec"].Event != "writeTest" {
		t.Fatalf("spec event = %q, want writeTest", guidance.States["spec"].Event)
	}
}

func TestDriveCommandRunsBuildJobToDoneWithFakeBackend(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)
	backend := &stubBackend{responses: []string{"start", "finish"}}
	cmd.backend = backend

	if err := cmd.run([]string{"drive", "build-job", "--task", "Drive build-job to done."}); err != nil {
		t.Fatalf("drive returned error: %v", err)
	}
	output := stdout.String()
	for _, want := range []string{
		"started: build-job",
		"advanced: idle -> running",
		"advanced: running -> done",
		"terminal: done",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("drive output missing %q:\n%s", want, output)
		}
	}
	if len(backend.prompts) != 2 {
		t.Fatalf("backend prompts = %d, want 2", len(backend.prompts))
	}
	for _, want := range []string{"Drive build-job to done.", "Current state: idle"} {
		if !strings.Contains(backend.prompts[0], want) {
			t.Fatalf("prompt 0 missing %q:\n%s", want, backend.prompts[0])
		}
	}
}

func TestSmokeDriveScriptExists(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "tests", "smoke", "circuit_drive_smoke.py")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected smoke script %s: %v", path, err)
	}
}

func TestUnknownCommandFails(t *testing.T) {
	t.Parallel()
	stderr := &bytes.Buffer{}
	cmd := command{stdout: &bytes.Buffer{}, stderr: stderr}

	err := cmd.run([]string{"nope"})

	if err == nil {
		t.Fatal("unknown command returned nil error")
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("unknown command did not print usage: %q", stderr.String())
	}
}

func testCommand(t *testing.T, stdout *bytes.Buffer) command {
	t.Helper()
	root := t.TempDir()
	machines := filepath.Join(root, "machines")
	if err := os.MkdirAll(machines, 0o700); err != nil {
		t.Fatalf("create machines dir: %v", err)
	}
	for _, name := range []string{"build-job.mch", "build-job.prompts.yaml", "pr-watch.mch", "review-flow.mch", "review-flow.checks.yaml", "retry-flow.mch", "retry-flow.checks.yaml", "tdd-flow.mch", "tdd-flow.checks.yaml", "tdd-flow.prompts.yaml", "check-registry.yaml", "alternating-check.sh"} {
		copyTestFile(t, filepath.Join("..", "..", "machines", name), filepath.Join(machines, name))
	}
	return command{stdout: stdout, stderr: &bytes.Buffer{}, cwd: root}
}

type advanceJSONTest struct {
	Allowed bool     `json:"allowed"`
	Event   string   `json:"event"`
	From    string   `json:"from"`
	To      string   `json:"to"`
	Failed  []string `json:"failed"`
}

type statusJSONTest struct {
	Session      string               `json:"session"`
	SessionState string               `json:"sessionState"`
	Machine      string               `json:"machine"`
	Current      string               `json:"current"`
	Enabled      []callStatusJSONTest `json:"enabled"`
	Blocked      []callStatusJSONTest `json:"blocked"`
}

type callStatusJSONTest struct {
	Call   string   `json:"call"`
	Failed []string `json:"failed"`
}

func assertStatusJSON(t *testing.T, reports []statusJSONTest, session string, state string, current string) {
	t.Helper()
	for _, report := range reports {
		if report.Session == session {
			if report.SessionState != state || report.Current != current {
				t.Fatalf("status for %s = %s/%s, want %s/%s", session, report.SessionState, report.Current, state, current)
			}
			if report.Machine != "build-job" {
				t.Fatalf("machine for %s = %s, want build-job", session, report.Machine)
			}
			if len(report.Enabled) == 0 && len(report.Blocked) == 0 {
				t.Fatalf("status for %s missing operations: %#v", session, report)
			}
			return
		}
	}
	t.Fatalf("missing status for session %s in %#v", session, reports)
}

func extractSessionID(t *testing.T, output string) string {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "session: ") {
			return strings.TrimPrefix(line, "session: ")
		}
	}
	t.Fatalf("output missing session id: %q", output)
	return ""
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
	content := []byte("checks:\n  makeCheck:\n    kind: command\n    command: " + command + "\n    returns: BOOL\n")
	path := filepath.Join(root, "machines", "check-registry.yaml")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write registry: %v", err)
	}
}

func writeTDDRegistry(t *testing.T, root string, testSuiteCommand string) {
	t.Helper()
	content := []byte("checks:\n  testSuitePassed:\n    kind: command\n    command: " + testSuiteCommand + "\n    returns: BOOL\n")
	path := filepath.Join(root, "machines", "check-registry.yaml")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write tdd registry: %v", err)
	}
}

func copyTestFile(t *testing.T, source string, destination string) {
	t.Helper()
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read test fixture %s: %v", source, err)
	}
	if err := os.WriteFile(destination, content, 0o600); err != nil {
		t.Fatalf("write test fixture %s: %v", destination, err)
	}
}
