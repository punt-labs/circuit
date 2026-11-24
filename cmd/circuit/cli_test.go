package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIRejectsRetiredPathsAndOptionsInJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, name := range []string{"list", "load", "scaffold", "start", "advance", "stop", "unload", "drive", "help", "-V", "--version", "machine check"} {
		t.Run(name, func(t *testing.T) {
			assertCLIFailureJSON(t, []string{name, "--json"})
		})
	}
	assertCLIFailureJSON(t, []string{"status", "--json", "old-session-argument"})
	assertCLIFailureJSON(t, []string{"--quiet", "machine", "list", "--verbose", "--json"})
	assertCLIFailureJSON(t, []string{"--json", "completion", "bash"})
	assertCLIFailureJSON(t, []string{"--json", "session", "drive", "--task"})
}

func assertCLIFailureJSON(t *testing.T, args []string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := runCLI(args, &stdout, &stderr); code == 0 {
		t.Fatalf("%v succeeded: %s", args, &stdout)
	}
	var result map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || result["error"] == "" {
		t.Fatalf("%v did not return one JSON error object: %s (%v)", args, &stdout, err)
	}
}

func TestGlobalFlagsAtEveryPosition(t *testing.T) {
	t.Parallel()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd := testCommand(t, stdout)
	cmd.stderr = stderr
	for _, args := range [][]string{
		{"--json", "--verbose", "machine", "list"},
		{"machine", "-v", "--json", "list"},
		{"machine", "list", "--json", "--verbose"},
	} {
		stdout.Reset()
		stderr.Reset()
		if err := cmd.run(args); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		var names []string
		if err := json.Unmarshal(stdout.Bytes(), &names); err != nil || len(names) == 0 || names[0] != "build-job" {
			t.Fatalf("%v: invalid machine list %q (%v)", args, stdout, err)
		}
		if !strings.Contains(stderr.String(), "machine list: succeeded") {
			t.Fatalf("%v did not emit diagnostics: %s", args, stderr)
		}
	}
}

func TestSessionAdvanceRejectsBlankEvent(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)
	if err := cmd.run([]string{"session", "start", "build-job"}); err != nil {
		t.Fatal(err)
	}
	id := extractSessionID(t, stdout.String())
	for _, event := range []string{"", " \t\n"} {
		stdout.Reset()
		err := cmd.run([]string{"session", "advance", event, id, "--json"})
		var result map[string]string
		if decodeErr := json.Unmarshal(stdout.Bytes(), &result); decodeErr != nil ||
			err == nil || !strings.Contains(result["error"], "nonempty event") {
			t.Fatalf("advance %q: output %q, error %v, decode error %v", event, stdout, err, decodeErr)
		}
	}
}

func TestSessionShowAndStatusJSON(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)
	if err := cmd.run([]string{"session", "start", "build-job"}); err != nil {
		t.Fatal(err)
	}
	id := extractSessionID(t, stdout.String())
	stdout.Reset()
	if err := cmd.run([]string{"session", "show", id, "--json"}); err != nil {
		t.Fatal(err)
	}
	var report statusJSONTest
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || report.Session != id {
		t.Fatalf("show JSON = %q (%v)", stdout, err)
	}
	stdout.Reset()
	if err := cmd.run([]string{"status", "--json"}); err != nil {
		t.Fatal(err)
	}
	var counts map[string]int
	if err := json.Unmarshal(stdout.Bytes(), &counts); err != nil || counts["active"] != 1 || counts["stopped"] != 0 {
		t.Fatalf("status JSON = %q (%v)", stdout, err)
	}
}

func TestSessionDriveSingleJSONResult(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)
	cmd.backend = &stubBackend{responses: []string{"start", "finish"}}
	if err := cmd.run([]string{"-q", "session", "drive", "build-job", "--task", "a goal", "--json"}); err != nil {
		t.Fatal(err)
	}
	var report statusJSONTest
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || report.Current != "done" || report.SessionState != "stopped" {
		t.Fatalf("drive JSON = %q (%v)", stdout, err)
	}
}

func TestDoctorReportsRequiredAndOptionalChecks(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	var stdout bytes.Buffer
	if code := runCLI([]string{"doctor", "--json"}, &stdout, io.Discard); code != 0 {
		t.Fatalf("doctor exit %d: %s", code, &stdout)
	}
	var report doctorReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || !report.OK || len(report.Checks) != 4 ||
		report.Checks[2].OK || report.Checks[2].Required {
		t.Fatalf("doctor JSON = %q (%v)", &stdout, err)
	}
	t.Setenv("HOME", "relative")
	stdout.Reset()
	if code := runCLI([]string{"--quiet", "doctor", "--json"}, &stdout, io.Discard); code == 0 {
		t.Fatal("doctor succeeded with invalid HOME")
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || report.OK || report.Failure == "" || report.Checks[1].OK {
		t.Fatalf("failed doctor JSON = %q (%v)", &stdout, err)
	}
}

func TestInstallCopiesRunningExecutable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for range 2 {
		var stdout bytes.Buffer
		if code := runCLI([]string{"install", "--json"}, &stdout, io.Discard); code != 0 {
			t.Fatalf("install exit %d: %s", code, &stdout)
		}
		var report struct{ Path string }
		path := filepath.Join(home, ".local", "bin", "circuit")
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || report.Path != path {
			t.Fatalf("install JSON = %q (%v)", &stdout, err)
		}
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) || err != nil || info.Mode().Perm()&0o111 == 0 || info.Size() == 0 {
			t.Fatalf("installed executable: %v (%v)", info, err)
		}
	}
}

func TestCompletionIncludesPublicTree(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	cmd := command{stdout: &stdout, stderr: io.Discard}
	if err := cmd.run([]string{"completion", "bash"}); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"choices='machine session version status doctor install completion'",
		"choices='list check scaffold'",
		"choices='start list show advance stop remove drive'",
		"options+=' --task'",
	} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Errorf("Bash completion missing %s", fragment)
		}
	}
}
