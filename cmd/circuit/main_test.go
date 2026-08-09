package main

import (
	"bytes"
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
	if !strings.Contains(output, "build-job") || !strings.Contains(output, "pr-watch") {
		t.Fatalf("list output mismatch: %q", output)
	}
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

func TestBMachineStop(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)

	if err := cmd.run([]string{"start", "build-job"}); err != nil {
		t.Fatalf("start returned error: %v", err)
	}
	stdout.Reset()
	if err := cmd.run([]string{"stop"}); err != nil {
		t.Fatalf("stop returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "stopped") {
		t.Fatalf("stop output mismatch: %q", stdout.String())
	}
	if err := cmd.run([]string{"status"}); err != nil {
		t.Fatalf("status after stop returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "no active session") {
		t.Fatalf("status after stop output mismatch: %q", stdout.String())
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

func TestStatusRejectsExtraArgs(t *testing.T) {
	t.Parallel()
	cmd := testCommand(t, &bytes.Buffer{})

	err := cmd.run([]string{"status", "extra"})

	if err == nil {
		t.Fatal("status with extra args returned nil error")
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

func TestStopRejectsExtraArgs(t *testing.T) {
	t.Parallel()
	cmd := testCommand(t, &bytes.Buffer{})

	err := cmd.run([]string{"stop", "extra"})

	if err == nil {
		t.Fatal("stop with extra args returned nil error")
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
	for _, name := range []string{"build-job.mch", "pr-watch.mch", "review-flow.mch", "review-flow.checks.yaml", "retry-flow.mch", "retry-flow.checks.yaml", "check-registry.yaml", "alternating-check.sh"} {
		copyTestFile(t, filepath.Join("..", "..", "machines", name), filepath.Join(machines, name))
	}
	return command{stdout: stdout, stderr: &bytes.Buffer{}, cwd: root}
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
