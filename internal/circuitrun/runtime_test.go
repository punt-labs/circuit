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
	if _, err := runtime.Start("missing"); err == nil {
		t.Fatal("start missing machine returned nil error")
	}
}

func TestRuntimeSuspendsAndResumes(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume empty runtime: %v", err)
	}
	if _, err := runtime.Start("build-job"); err != nil {
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
	if len(machines) != 2 || machines[0] != "build-job" || machines[1] != "review-flow" {
		t.Fatalf("machines = %v, want build-job/review-flow", machines)
	}
	if runtime.SuspendedPath() != filepath.Join(root, ".tmp", "circuit.suspended.json") {
		t.Fatalf("suspended path = %s", runtime.SuspendedPath())
	}
}

func TestRuntimeStopClearsSuspendedRuntime(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume empty runtime: %v", err)
	}
	if _, err := runtime.Start("build-job"); err != nil {
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
	if _, err := resumed.Status(); err == nil {
		t.Fatal("status after stop returned nil error")
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
	if _, err := runtime.Start("review-flow"); err != nil {
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
	if _, err := runtime.Start("review-flow"); err != nil {
		t.Fatalf("start review-flow: %v", err)
	}
	_, err = runtime.Advance("requestReview")
	if err == nil || !strings.Contains(err.Error(), "unknown registry") {
		t.Fatalf("advance error = %v, want unknown registry", err)
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
	if _, err := runtime.Start("review-flow"); err != nil {
		t.Fatalf("start review-flow: %v", err)
	}
	_, err = runtime.Advance("requestReview")
	if err == nil || !strings.Contains(err.Error(), "returning BOOL") {
		t.Fatalf("advance error = %v, want BOOL registry error", err)
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
	if _, err := runtime.Start("review-flow"); err != nil {
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

func testRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	machines := filepath.Join(root, "machines")
	if err := os.MkdirAll(machines, 0o700); err != nil {
		t.Fatalf("create machines dir: %v", err)
	}
	for _, name := range []string{"build-job.mch", "review-flow.mch", "review-flow.checks.yaml", "check-registry.yaml"} {
		copyFixture(t, filepath.Join("..", "..", "machines", name), filepath.Join(machines, name))
	}
	return root
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
