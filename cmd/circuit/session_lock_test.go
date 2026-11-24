package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/punt-labs/circuit/internal/circuitrun"
)

func TestAdvanceWithNestedCircuitCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, checkArgs, wantCurrent, wantError string
	}{
		{"status", "status --json", "done", ""},
		{"text-status", "status", "done", ""},
		{"stop", `stop "$CIRCUIT_SESSION_ID"`, "waiting", "changed during advance; retry the command"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := testCommand(t, &bytes.Buffer{})
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			check := fmt.Sprintf("checks:\n  alternatingCheck:\n    kind: command\n    returns: BOOL\n    command: >-\n      '%s' -test.run=^TestSessionCLIProcess$ -- %s\n",
				strings.ReplaceAll(binary, "'", `'"'"'`), tc.checkArgs)
			registry := filepath.Join(cmd.cwd, "machines", "check-registry.yaml")
			if err := os.WriteFile(registry, []byte(check), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := cmd.run([]string{"start", "retry-flow"}); err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			process := exec.CommandContext(ctx, binary, "-test.run=^TestSessionCLIProcess$", "--", "advance", "proceed")
			process.Dir = cmd.cwd
			process.Env = append(os.Environ(), "CIRCUIT_TEST_SESSION_CLI=1", "HOME="+t.TempDir(), "CIRCUIT_TELEMETRY=0")
			output, err := process.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatalf("nested circuit command timed out: %v", ctx.Err())
			}
			if tc.wantError == "" && err != nil || tc.wantError != "" && (err == nil || !strings.Contains(string(output), tc.wantError)) {
				t.Fatalf("advance: %v; output: %s", err, output)
			}

			stdout := &bytes.Buffer{}
			cmd.stdout = stdout
			if err := cmd.run([]string{"status", "--json"}); err != nil {
				t.Fatal(err)
			}
			var reports []statusJSONTest
			if err := json.Unmarshal(stdout.Bytes(), &reports); err != nil {
				t.Fatal(err)
			}
			if len(reports) != 1 || reports[0].Current != tc.wantCurrent || reports[0].SessionState != "stopped" {
				t.Fatalf("unexpected final session: %+v", reports)
			}
		})
	}
}

func TestSessionCLIProcess(t *testing.T) {
	if os.Getenv("CIRCUIT_TEST_SESSION_CLI") != "1" {
		return
	}
	idx := slices.Index(os.Args, "--")
	if idx < 0 {
		t.Fatal("missing argument separator")
	}
	os.Exit(runCLI(os.Args[idx+1:], os.Stdout, os.Stderr))
}

func TestListLoadAndJSONStatusInReadOnlyWorkspace(t *testing.T) {
	t.Parallel()
	for _, withSession := range []bool{false, true} {
		t.Run(fmt.Sprintf("session=%t", withSession), func(t *testing.T) {
			stdout := &bytes.Buffer{}
			cmd := testCommand(t, stdout)
			stateRoot := filepath.Join(cmd.cwd, ".punt-labs", "local", "circuit")
			dirs := []string{cmd.cwd}
			wantSessions := 0
			if withSession {
				if err := cmd.run([]string{"start", "build-job"}); err != nil {
					t.Fatal(err)
				}
				dirs = append(dirs, stateRoot, filepath.Join(stateRoot, "sessions"))
				wantSessions = 1
			}
			for _, dir := range dirs {
				if err := os.Chmod(dir, 0o500); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := os.Chmod(dir, 0o700); err != nil {
						t.Error(err)
					}
				})
			}
			for _, args := range [][]string{{"list"}, {"load", "build-job"}, {"status", "--json"}} {
				stdout.Reset()
				if err := cmd.run(args); err != nil {
					t.Fatalf("%v: %v", args, err)
				}
			}
			var reports []statusJSONTest
			if err := json.Unmarshal(stdout.Bytes(), &reports); err != nil || len(reports) != wantSessions {
				t.Fatalf("expected %d sessions, got %s (decode: %v)", wantSessions, stdout, err)
			}
			if !withSession {
				if _, err := os.Stat(stateRoot); !os.IsNotExist(err) {
					t.Fatalf("read-only commands created the private store: %v", err)
				}
			}
		})
	}
}

func TestOpenDriveTraceDoesNotOverwriteExistingTrace(t *testing.T) {
	t.Parallel()
	cmd := testCommand(t, &bytes.Buffer{})
	runtime := circuitrun.New(cmd.cwd)
	id := "build-job-abcd"
	trace, err := cmd.openDriveTrace(runtime, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := trace.Write([]byte("original trace")); err != nil {
		t.Fatal(err)
	}
	if err := trace.Close(); err != nil {
		t.Fatal(err)
	}
	if duplicate, err := cmd.openDriveTrace(runtime, id); !os.IsExist(err) {
		if duplicate != nil {
			_ = duplicate.Close()
		}
		t.Fatalf("reopen trace: got %v, want file exists", err)
	}
	path := filepath.Join(cmd.cwd, ".punt-labs", "local", "circuit", "traces", id, "drive.jsonl")
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "original trace" {
		t.Fatalf("original trace was not preserved: %q (%v)", content, err)
	}
}

func TestConcurrentScaffoldPreservesRegistryEntries(t *testing.T) {
	t.Parallel()
	cmd := testCommand(t, &bytes.Buffer{})
	for _, name := range []string{"check-registry.yaml", "retry-flow.checks.yaml", "review-flow.checks.yaml"} {
		if err := os.Remove(filepath.Join(cmd.cwd, "machines", name)); err != nil {
			t.Fatal(err)
		}
	}
	machines := []string{"retry-flow", "review-flow"}
	start := make(chan struct{})
	var workers sync.WaitGroup
	for _, machine := range machines {
		workers.Go(func() {
			worker := cmd
			worker.stdout = &bytes.Buffer{}
			<-start
			if err := worker.run([]string{"scaffold", machine}); err != nil {
				t.Errorf("scaffold %s: %v", machine, err)
			}
		})
	}
	close(start)
	workers.Wait()

	for _, machine := range machines {
		if err := cmd.run([]string{"load", machine}); err != nil {
			t.Errorf("load %s after concurrent scaffold: %v", machine, err)
		}
	}
}
