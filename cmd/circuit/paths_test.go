package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/punt-labs/circuit/internal/circuitpath"
)

func TestResolveWorkspaceOutsideGit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for range 2 {
		root := t.TempDir()
		workspace, err := circuitpath.ResolveWorkspace(root)
		if err != nil {
			t.Fatal(err)
		}
		canonical, err := filepath.EvalSymlinks(root)
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(home, ".punt-labs", "circuit", "data", "workspaces", fmt.Sprintf("%x", sha256.Sum256([]byte(canonical))))
		if workspace.StateRoot != want || workspace.LockDir != canonical {
			t.Fatalf("workspace = %+v, want state %q and lock %q", workspace, want, canonical)
		}
	}
}

func TestResolveWorkspaceWithoutAbsoluteHome(t *testing.T) {
	for _, home := range []string{"", "relative"} {
		t.Setenv("HOME", home)
		if workspace, err := circuitpath.ResolveWorkspace(t.TempDir()); err == nil {
			t.Fatalf("HOME=%q: resolved workspace %+v without a valid home", home, workspace)
		}
	}
}

func TestNonGitWorkspacePersistsSessionsUnderHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)
	if err := os.Remove(filepath.Join(cmd.cwd, ".git")); err != nil {
		t.Fatal(err)
	}
	if err := cmd.run([]string{"start", "build-job"}); err != nil {
		t.Fatal(err)
	}
	id := extractSessionID(t, stdout.String())
	matches, err := filepath.Glob(filepath.Join(home, ".punt-labs", "circuit", "data", "workspaces", "*", "sessions", id+".json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("session files = %v, err = %v", matches, err)
	}
	stdout.Reset()
	if err := cmd.run([]string{"status", "--json", id}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), id) {
		t.Fatalf("saved session missing from status: %s", stdout)
	}
	if _, err := os.Stat(filepath.Join(cmd.cwd, ".tmp")); !os.IsNotExist(err) {
		t.Fatalf("runtime wrote to the old state directory: %v", err)
	}
}

func TestResolveRepositoryAndLinkedWorktree(t *testing.T) {
	t.Setenv("HOME", "")
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	checkout := filepath.Join(root, "linked")
	store := filepath.Join(root, "main")
	if err := os.Mkdir(store, 0o700); err != nil {
		t.Fatal(err)
	}
	git(t, store, "init", "-q")
	git(t, store, "-c", "user.name=Circuit", "-c", "user.email=circuit@example.invalid", "commit", "--allow-empty", "-qm", "Initial commit")
	git(t, store, "worktree", "add", "--detach", checkout)
	path := os.Getenv("PATH")
	t.Setenv("PATH", "")
	local := filepath.Join(store, ".punt-labs", "local", "circuit")
	linkedLocal := filepath.Join(local, "worktrees", fmt.Sprintf("%x", sha256.Sum256([]byte(checkout))))
	for _, c := range []struct {
		checkout string
		local    string
	}{{store, local}, {checkout, linkedLocal}} {
		workspace, err := circuitpath.ResolveWorkspace(c.checkout)
		if err != nil || workspace.StateRoot != c.local || workspace.LockDir != c.checkout {
			t.Fatalf("checkout workspace = %+v, err = %v", workspace, err)
		}
		nested := filepath.Join(c.checkout, "nested")
		if err := os.Mkdir(nested, 0o700); err != nil {
			t.Fatal(err)
		}
		workspace, err = circuitpath.ResolveWorkspace(nested)
		want := filepath.Join(c.local, "workspaces", fmt.Sprintf("%x", sha256.Sum256([]byte(nested))))
		if err != nil || workspace.StateRoot != want || workspace.LockDir != nested {
			t.Fatalf("nested workspace = %+v, err = %v", workspace, err)
		}
	}
	if err := os.MkdirAll(linkedLocal, 0o700); err != nil {
		t.Fatal(err)
	}
	trace := filepath.Join(linkedLocal, "trace")
	if err := os.WriteFile(trace, []byte("retained"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", path)
	git(t, store, "worktree", "remove", "--force", checkout)
	if content, err := os.ReadFile(trace); err != nil || string(content) != "retained" {
		t.Fatalf("linked trace was not preserved: %q (%v)", content, err)
	}
}

func TestStateIsIgnoredInGitWithoutAnExistingIgnoreRule(t *testing.T) {
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)
	git(t, cmd.cwd, "init", "-q")
	cmd.backend = &stubBackend{responses: []string{"start", "finish"}}
	if err := cmd.run([]string{"drive", "build-job"}); err != nil {
		t.Fatal(err)
	}
	id := extractSessionID(t, stdout.String())
	for _, path := range []string{
		filepath.Join(".punt-labs", "local", "circuit", "sessions", id+".json"),
		filepath.Join(".punt-labs", "local", "circuit", "traces", id, "drive.jsonl"),
	} {
		if _, err := os.Stat(filepath.Join(cmd.cwd, path)); err != nil {
			t.Fatal(err)
		}
		git(t, cmd.cwd, "check-ignore", "-q", path)
	}
	if status := git(t, cmd.cwd, "status", "--short", "--", ".punt-labs"); status != "" {
		t.Fatalf("private state visible in Git status: %s", status)
	}
}

func TestStateIgnoreDoesNotReplaceExistingFile(t *testing.T) {
	cmd := testCommand(t, &bytes.Buffer{})
	workspace, err := circuitpath.ResolveWorkspace(cmd.cwd)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(workspace.StateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(workspace.StateRoot, ".gitignore")
	if err := os.WriteFile(path, []byte("traces/\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cmd.run([]string{"start", "build-job"}); err == nil || !strings.Contains(err.Error(), "private state ignore file") {
		t.Fatalf("start with incompatible ignore file: %v", err)
	}
	if contents, err := os.ReadFile(path); err != nil || string(contents) != "traces/\n" {
		t.Fatalf("existing ignore file was changed: %q (%v)", contents, err)
	}
	if _, err := os.Stat(filepath.Join(workspace.StateRoot, "sessions")); !os.IsNotExist(err) {
		t.Fatalf("private session state written without an ignore rule: %v", err)
	}
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
	return string(output)
}
