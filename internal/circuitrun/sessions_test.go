package circuitrun

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldNoBooleanVariablesIsNoop(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}

	report, err := runtime.Scaffold("build-job")

	if err != nil {
		t.Fatalf("scaffold build-job: %v", err)
	}
	if len(report.GeneratedBindings) != 0 || len(report.GeneratedRegistryIDs) != 0 {
		t.Fatalf("scaffold build-job = %#v, want no generated checks", report)
	}
	load, err := runtime.Load("build-job")
	if err != nil {
		t.Fatalf("load build-job: %v", err)
	}
	if len(load.Checks) != 0 {
		t.Fatalf("load build-job checks = %v, want none", load.Checks)
	}
	if _, err := os.Stat(filepath.Join(root, "machines", "build-job.checks.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("scaffold build-job check file err = %v, want not exist", err)
	}
}

func TestLoadFailsWhenBooleanChecksMissing(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	if err := os.Remove(filepath.Join(root, "machines", "review-flow.checks.yaml")); err != nil {
		t.Fatalf("remove bindings: %v", err)
	}
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}

	_, err = runtime.Load("review-flow")

	if err == nil || !strings.Contains(err.Error(), "unbound BOOL variable makeCheckPassed") {
		t.Fatalf("load error = %v, want unbound BOOL variable", err)
	}
}

func TestStartFailsWhenBooleanChecksMissing(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	if err := os.Remove(filepath.Join(root, "machines", "review-flow.checks.yaml")); err != nil {
		t.Fatalf("remove bindings: %v", err)
	}
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}

	_, _, err = runtime.Start("review-flow")

	if err == nil || !strings.Contains(err.Error(), "unbound BOOL variable makeCheckPassed") {
		t.Fatalf("start error = %v, want unbound BOOL variable", err)
	}
}

func TestScaffoldGeneratesMissingBindingsAndFalseStubs(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	if err := os.Remove(filepath.Join(root, "machines", "review-flow.checks.yaml")); err != nil {
		t.Fatalf("remove bindings: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "machines", "check-registry.yaml"), []byte("checks:\n  existing:\n    kind: command\n    command: true\n    returns: BOOL\n"), 0o600); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}

	report, err := runtime.Scaffold("review-flow")

	if err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	if len(report.GeneratedBindings) != 1 || report.GeneratedBindings[0] != "makeCheckPassed" {
		t.Fatalf("generated bindings = %v, want makeCheckPassed", report.GeneratedBindings)
	}
	bindings, err := os.ReadFile(filepath.Join(root, "machines", "review-flow.checks.yaml"))
	if err != nil {
		t.Fatalf("read bindings: %v", err)
	}
	if !strings.Contains(string(bindings), "makeCheckPassed:") || !strings.Contains(string(bindings), "use: makeCheckPassed") {
		t.Fatalf("bindings content = %q", string(bindings))
	}
	registry, err := os.ReadFile(filepath.Join(root, "machines", "check-registry.yaml"))
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	registryText := string(registry)
	if !strings.Contains(registryText, "existing:") {
		t.Fatalf("registry overwrote existing entry: %q", registryText)
	}
	if !strings.Contains(registryText, "makeCheckPassed:") || !strings.Contains(registryText, "command: \"false\"") {
		t.Fatalf("registry missing false stub: %q", registryText)
	}
}

func TestScaffoldStubsBlockUntilReplaced(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	if err := os.Remove(filepath.Join(root, "machines", "review-flow.checks.yaml")); err != nil {
		t.Fatalf("remove bindings: %v", err)
	}
	if err := os.Remove(filepath.Join(root, "machines", "check-registry.yaml")); err != nil {
		t.Fatalf("remove registry: %v", err)
	}
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if _, err := runtime.Scaffold("review-flow"); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	if _, _, err := runtime.Start("review-flow"); err != nil {
		t.Fatalf("start with scaffolded stubs: %v", err)
	}
	blocked, err := runtime.Advance("requestReview")
	if err != nil {
		t.Fatalf("advance with false stub: %v", err)
	}
	if blocked.Allowed {
		t.Fatal("false stub allowed requestReview")
	}

	content := []byte("checks:\n  makeCheckPassed:\n    kind: command\n    command: true\n    returns: BOOL\n")
	if err := os.WriteFile(filepath.Join(root, "machines", "check-registry.yaml"), content, 0o600); err != nil {
		t.Fatalf("replace scaffolded registry: %v", err)
	}
	allowed, err := runtime.Advance("requestReview")
	if err != nil {
		t.Fatalf("advance with true replacement: %v", err)
	}
	if !allowed.Allowed || allowed.To != "codeReview" {
		t.Fatalf("advance with real command = %#v, want codeReview", allowed)
	}
}

func TestStartCreatesSessionWithMachineHexID(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	id, _, err := runtime.Start("build-job")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if !strings.HasPrefix(id, "build-job-") || len(id) != len("build-job-")+4 {
		t.Fatalf("session id = %q, want build-job-XXXX", id)
	}
}

func TestTwoSessionsSameMachine(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	id1, _, err := runtime.Start("build-job")
	if err != nil {
		t.Fatalf("start 1: %v", err)
	}
	id2, _, err := runtime.Start("build-job")
	if err != nil {
		t.Fatalf("start 2: %v", err)
	}
	if id1 == id2 {
		t.Fatalf("duplicate session ids: %s", id1)
	}
}

func TestStatusAllShowsAllActiveSessions(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	runtime.Start("build-job")
	runtime.Start("build-job")
	all, err := runtime.StatusAll()
	if err != nil {
		t.Fatalf("StatusAll: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("StatusAll = %d sessions, want 2", len(all))
	}
}

func TestStatusByID(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	id, _, _ := runtime.Start("build-job")
	report, err := runtime.StatusByID(id)
	if err != nil {
		t.Fatalf("status by id: %v", err)
	}
	if report.MachineName != "build-job" || report.Current != "idle" {
		t.Fatalf("status = %s/%s", report.MachineName, report.Current)
	}
}

func TestStatusByIDUnknown(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	_, err = runtime.StatusByID("build-job-0000")
	if err == nil {
		t.Fatal("status unknown id returned nil error")
	}
}

func TestAdvanceImplicitWithOneSession(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	runtime.Start("build-job")
	report, err := runtime.Advance("start")
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if !report.Allowed || report.To != "running" {
		t.Fatalf("advance = %#v", report)
	}
}

func TestAdvanceAmbiguousWithMultipleSessions(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	runtime.Start("build-job")
	runtime.Start("build-job")
	_, err = runtime.Advance("start")
	if err == nil {
		t.Fatal("advance with multiple sessions returned nil error")
	}
	if !strings.Contains(err.Error(), "multiple active sessions") {
		t.Fatalf("advance error = %v", err)
	}
}

func TestAdvanceByID(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	id, _, _ := runtime.Start("build-job")
	report, err := runtime.AdvanceByID(id, "start")
	if err != nil {
		t.Fatalf("advance by id: %v", err)
	}
	if !report.Allowed || report.To != "running" {
		t.Fatalf("advance = %#v", report)
	}
}

func TestStopByIDLeavesOtherSessionActive(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	id1, _, _ := runtime.Start("build-job")
	id2, _, _ := runtime.Start("build-job")
	if err := runtime.StopByID(id1); err != nil {
		t.Fatalf("stop: %v", err)
	}
	all, err := runtime.StatusAll()
	if err != nil {
		t.Fatalf("StatusAll: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("StatusAll after stop = %d, want 1", len(all))
	}
	if all[0].SessionID != id2 {
		t.Fatalf("remaining session = %s, want %s", all[0].SessionID, id2)
	}
}

func TestAutoStopPerSession(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	id1, _, _ := runtime.Start("build-job")
	id2, _, _ := runtime.Start("build-job")

	// Advance id1 to terminal
	runtime.AdvanceByID(id1, "start")
	runtime.AdvanceByID(id1, "finish")

	all, err := runtime.StatusAll()
	if err != nil {
		t.Fatalf("StatusAll: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("StatusAll after auto-stop = %d, want 1", len(all))
	}
	if all[0].SessionID != id2 {
		t.Fatalf("remaining session = %s, want %s", all[0].SessionID, id2)
	}
}

func TestSuspendAndResumeMultipleSessions(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	id1, _, _ := runtime.Start("build-job")
	runtime.Start("build-job")
	runtime.AdvanceByID(id1, "start")
	if err := runtime.Suspend(); err != nil {
		t.Fatalf("suspend: %v", err)
	}

	resumed, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	all, err := resumed.StatusAll()
	if err != nil {
		t.Fatalf("StatusAll: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("resumed StatusAll = %d, want 2", len(all))
	}
	// Verify id1 preserved its advanced state
	for _, s := range all {
		if s.SessionID == id1 && s.Current != "running" {
			t.Fatalf("resumed session %s current = %s, want running", id1, s.Current)
		}
	}
}

func TestNoActiveSessionsStatusAllEmpty(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	all, err := runtime.StatusAll()
	if err != nil {
		t.Fatalf("StatusAll: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("StatusAll empty = %d, want 0", len(all))
	}
}

func TestResumeRejectsMalformedSessionFile(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	path := filepath.Join(root, ".tmp", "sessions", "build-job-0000.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create sessions dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("write malformed session: %v", err)
	}

	_, err := Resume(root)

	if err == nil {
		t.Fatal("resume malformed session returned nil error")
	}
}

func TestResumeUsesFilenameWhenSessionIDMissing(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	id, _, err := runtime.Start("build-job")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	path := runtime.sessionPath(id)
	if err := runtime.Suspend(); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read session: %v", err)
	}
	withoutID := strings.Replace(string(content), "  \"sessionId\": \""+id+"\",\n", "", 1)
	if err := os.WriteFile(path, []byte(withoutID), 0o600); err != nil {
		t.Fatalf("rewrite session without id: %v", err)
	}

	resumed, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	status, err := resumed.StatusByID(id)
	if err != nil {
		t.Fatalf("status by filename-derived id: %v", err)
	}
	if status.SessionID != id {
		t.Fatalf("status session = %s, want %s", status.SessionID, id)
	}
}

func TestSuspendRemovesStaleSessionFiles(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	id, _, err := runtime.Start("build-job")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := runtime.Suspend(); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	delete(runtime.sessions, id)
	if err := runtime.Suspend(); err != nil {
		t.Fatalf("suspend after delete: %v", err)
	}
	if _, err := os.Stat(runtime.sessionPath(id)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale session file err = %v, want not exist", err)
	}
}

func TestStopWithNoSessionsClearsLegacySuspendedFile(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(runtime.suspendedPath()), 0o700); err != nil {
		t.Fatalf("create legacy dir: %v", err)
	}
	if err := os.WriteFile(runtime.suspendedPath(), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write legacy suspended file: %v", err)
	}

	if err := runtime.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if _, err := os.Stat(runtime.suspendedPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy suspended file err = %v, want not exist", err)
	}
}

func TestScaffoldFullPathWritesBesideMachine(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	if err := os.Remove(filepath.Join(root, "machines", "review-flow.checks.yaml")); err != nil {
		t.Fatalf("remove bindings: %v", err)
	}
	if err := os.Remove(filepath.Join(root, "machines", "check-registry.yaml")); err != nil {
		t.Fatalf("remove registry: %v", err)
	}
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	machinePath := filepath.Join(root, "machines", "review-flow.mch")

	if _, err := runtime.Scaffold(machinePath); err != nil {
		t.Fatalf("scaffold full path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "machines", "review-flow.checks.yaml")); err != nil {
		t.Fatalf("full path scaffold bindings: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "machines", "check-registry.yaml")); err != nil {
		t.Fatalf("full path scaffold registry: %v", err)
	}
	if _, err := runtime.Load(machinePath); err != nil {
		t.Fatalf("load full path after scaffold: %v", err)
	}
}

func TestFullPathMachineSessionIDUsesBaseName(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	id, _, err := runtime.Start(filepath.Join(root, "machines", "build-job.mch"))
	if err != nil {
		t.Fatalf("start full path: %v", err)
	}
	if !strings.HasPrefix(id, "build-job-") {
		t.Fatalf("full path session id = %s, want build-job-*", id)
	}
}

func TestResumeMigratesLegacyActiveSession(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	legacy := Run{
		MachineName: "build-job",
		MachineFile: filepath.Join(root, "machines", "build-job.mch"),
		Current:     "idle",
		Session:     SessionActive,
		Booleans:    map[string]bool{},
		Checks:      map[string]CheckRuntime{},
	}
	content, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy run: %v", err)
	}
	path := filepath.Join(root, ".tmp", "circuit.suspended.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create legacy dir: %v", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write legacy run: %v", err)
	}

	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume legacy run: %v", err)
	}
	all, err := runtime.StatusAll()
	if err != nil {
		t.Fatalf("StatusAll: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("legacy StatusAll = %d, want 1", len(all))
	}
	if !strings.HasPrefix(all[0].SessionID, "build-job-") {
		t.Fatalf("legacy session id = %s, want build-job-*", all[0].SessionID)
	}
}

func TestResumeIgnoresLegacyStoppedSession(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	legacy := Run{MachineName: "build-job", Session: SessionStopped}
	content, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy run: %v", err)
	}
	path := filepath.Join(root, ".tmp", "circuit.suspended.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create legacy dir: %v", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write legacy run: %v", err)
	}

	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume legacy stopped run: %v", err)
	}
	if runtime.IsActive() {
		t.Fatal("legacy stopped session should not resume as active")
	}
}

func TestRuntimeErrorPaths(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if _, err := runtime.AdvanceByID("build-job-0000", "start"); err == nil {
		t.Fatal("advance unknown session returned nil error")
	}
	if _, err := runtime.ListMachines(); err != nil {
		t.Fatalf("list machines before removal: %v", err)
	}
	if err := os.Rename(filepath.Join(root, "machines"), filepath.Join(root, "machines.moved")); err != nil {
		t.Fatalf("move machines dir: %v", err)
	}
	if _, err := runtime.ListMachines(); err == nil {
		t.Fatal("list machines without directory returned nil error")
	}
}

func TestStatusFailsWhenMachineFileDisappears(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	id, _, err := runtime.Start("build-job")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := os.Rename(filepath.Join(root, "machines", "build-job.mch"), filepath.Join(root, "machines", "build-job.moved")); err != nil {
		t.Fatalf("move machine file: %v", err)
	}
	if _, err := runtime.StatusByID(id); err == nil {
		t.Fatal("status missing machine file returned nil error")
	}
	if _, err := runtime.StatusAll(); err == nil {
		t.Fatal("status all missing machine file returned nil error")
	}
}

func TestStopByIDRejectsUnknownSession(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if err := runtime.StopByID("build-job-0000"); err == nil {
		t.Fatal("stop unknown session returned nil error")
	}
}

func TestMalformedCheckFilesReturnErrors(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if _, _, err := runtime.Start("review-flow"); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "machines", "review-flow.checks.yaml"), []byte("checks: ["), 0o600); err != nil {
		t.Fatalf("write malformed bindings: %v", err)
	}
	if _, err := runtime.Advance("requestReview"); err == nil {
		t.Fatal("advance with malformed bindings returned nil error")
	}
	if err := os.WriteFile(filepath.Join(root, "machines", "review-flow.checks.yaml"), []byte("checks:\n  makeCheckPassed:\n    use: makeCheck\n"), 0o600); err != nil {
		t.Fatalf("restore bindings: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "machines", "check-registry.yaml"), []byte("checks: ["), 0o600); err != nil {
		t.Fatalf("write malformed registry: %v", err)
	}
	if _, err := runtime.Advance("requestReview"); err == nil {
		t.Fatal("advance with malformed registry returned nil error")
	}
}
