package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
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
	if err := cmd.run([]string{"session", "start", "build-job"}); err != nil {
		t.Fatal(err)
	}
	id := extractSessionID(t, stdout.String())
	matches, err := filepath.Glob(filepath.Join(home, ".punt-labs", "circuit", "data", "workspaces", "*", "sessions", id+".json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("session files = %v, err = %v", matches, err)
	}
	stdout.Reset()
	if err := cmd.run([]string{"session", "show", "--json", id}); err != nil {
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
	if workspace, err := circuitpath.ResolveWorkspace(checkout); err == nil || workspace.StateRoot != "" {
		t.Fatalf("resolved ambiguous linked workspace without Git: %+v, err = %v", workspace, err)
	}
	local := filepath.Join(store, ".punt-labs", "local", "circuit")
	linkedLocal := filepath.Join(local, "worktrees", fmt.Sprintf("%x", sha256.Sum256([]byte(checkout))))
	for _, c := range []struct {
		checkout string
		local    string
	}{{store, local}, {checkout, linkedLocal}} {
		if c.checkout == checkout {
			t.Setenv("PATH", path)
		}
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
	if err := os.Remove(filepath.Join(store, ".git", "config")); err != nil {
		t.Fatal(err)
	}
	git(t, checkout, "rev-parse", "--git-common-dir")
	if workspace, err := circuitpath.ResolveWorkspace(checkout); err != nil || workspace.StateRoot != linkedLocal {
		t.Fatalf("linked workspace without Git config = %+v, err = %v", workspace, err)
	}
	git(t, store, "worktree", "remove", "--force", checkout)
	if content, err := os.ReadFile(trace); err != nil || string(content) != "retained" {
		t.Fatalf("linked trace was not preserved: %q (%v)", content, err)
	}
}

func TestResolveLinkedWorktreeOfBareDotGitRepository(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	git(t, source, "init", "-q")
	git(t, source, "-c", "user.name=Circuit", "-c", "user.email=circuit@example.invalid", "commit", "--allow-empty", "-qm", "Initial commit")
	bare := filepath.Join(root, ".git")
	checkout := filepath.Join(root, "linked")
	git(t, source, "clone", "--bare", source, bare)
	git(t, bare, "worktree", "add", "--detach", checkout)
	workspace, err := circuitpath.ResolveWorkspace(checkout)
	want := filepath.Join(bare, ".punt-labs", "local", "circuit", "worktrees", fmt.Sprintf("%x", sha256.Sum256([]byte(checkout))))
	if err != nil || workspace.StateRoot != want {
		t.Fatalf("bare linked worktree = %+v, want state %q, err = %v", workspace, want, err)
	}
	config := filepath.Join(bare, "config")
	key := filepath.Base(want)
	for _, tc := range []struct {
		name      string
		config    string
		worktree  string
		wantOwner string
	}{
		{name: "worktree config", config: "[core]\n\trepositoryformatversion = 1\n\tbare = false\n[extensions]\n\tworktreeConfig = true\n", worktree: "[core]\n\tbare = true\n", wantOwner: bare},
		{name: "hex boolean", config: "[core]\n\tbare = 0x0\n", wantOwner: root},
		{name: "continued setting", config: "[core]\n\tbare = true\n\tdescription = text\\\nbare = false\n", wantOwner: bare},
		{name: "invalid boolean", config: "[core]\n\tbare = invalid\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(config, []byte(tc.config), 0o600); err != nil {
				t.Fatal(err)
			}
			if tc.worktree != "" {
				path := filepath.Join(bare, "config.worktree")
				if err := os.WriteFile(path, []byte(tc.worktree), 0o600); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := os.Remove(path); err != nil {
						t.Error(err)
					}
				})
			}
			workspace, err := circuitpath.ResolveWorkspace(checkout)
			if tc.wantOwner == "" {
				if err == nil || workspace.StateRoot != "" {
					t.Fatalf("resolved linked workspace with invalid Git config: %+v, err = %v", workspace, err)
				}
				return
			}
			want := filepath.Join(tc.wantOwner, ".punt-labs", "local", "circuit", "worktrees", key)
			if err != nil || workspace.StateRoot != want {
				t.Fatalf("linked workspace = %+v, want state %q, err = %v", workspace, want, err)
			}
		})
	}
}

func TestResolveCheckoutWithSeparateGitDirectory(t *testing.T) {
	checkout, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	git(t, checkout, "init", "--separate-git-dir", filepath.Join(t.TempDir(), "metadata"))
	workspace, err := circuitpath.ResolveWorkspace(checkout)
	want := filepath.Join(checkout, ".punt-labs", "local", "circuit")
	if err != nil || workspace.StateRoot != want {
		t.Fatalf("checkout with separate Git directory = %+v, want state %q, err = %v", workspace, want, err)
	}
}

func TestResolveWorkspaceRejectsNonregularGitMarker(t *testing.T) {
	checkout := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(checkout, ".git"), 0o600); err != nil {
		t.Fatal(err)
	}
	if workspace, err := circuitpath.ResolveWorkspace(checkout); err == nil || !strings.Contains(err.Error(), "invalid Git marker") {
		t.Fatalf("nonregular Git marker = %+v, err = %v", workspace, err)
	}
}

func TestStateIsIgnoredInGitWithoutAnExistingIgnoreRule(t *testing.T) {
	stdout := &bytes.Buffer{}
	cmd := testCommand(t, stdout)
	git(t, cmd.cwd, "init", "-q")
	cmd.backend = &stubBackend{responses: []string{"session", "start", "finish"}}
	if err := cmd.run([]string{"session", "drive", "build-job"}); err != nil {
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
	if err := cmd.run([]string{"session", "start", "build-job"}); err == nil || !strings.Contains(err.Error(), "private state ignore file") {
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
