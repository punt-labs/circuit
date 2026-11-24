package circuitpath

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const ToolName = "circuit"

func HomeRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve Circuit home: %w", err)
	}
	if !filepath.IsAbs(home) {
		return "", errors.New("set HOME to an absolute directory")
	}
	return filepath.Join(home, ".punt-labs", ToolName), nil
}

type Workspace struct {
	StateRoot string
	LockDir   string
	git       bool
}

func ResolveWorkspace(root string) (Workspace, error) {
	workspace, err := filepath.Abs(root)
	if err != nil {
		return Workspace{}, fmt.Errorf("resolve workspace: %w", err)
	}
	workspace, err = filepath.EvalSymlinks(workspace)
	if err != nil {
		return Workspace{}, fmt.Errorf("resolve workspace: %w", err)
	}
	for dir := workspace; ; dir = filepath.Dir(dir) {
		marker, err := os.Stat(filepath.Join(dir, ".git"))
		if err == nil {
			return repositoryWorkspace(dir, workspace, marker)
		}
		if !errors.Is(err, os.ErrNotExist) {
			return Workspace{}, fmt.Errorf("inspect Git workspace: %w", err)
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	home, err := HomeRoot()
	if err != nil {
		return Workspace{}, err
	}
	return Workspace{StateRoot: filepath.Join(home, "data", "workspaces", workspaceKey(workspace)), LockDir: workspace}, nil
}

func repositoryWorkspace(checkout, workspace string, marker os.FileInfo) (Workspace, error) {
	local := filepath.Join(".punt-labs", "local", ToolName)
	state := filepath.Join(checkout, local)
	if !marker.IsDir() {
		if !marker.Mode().IsRegular() {
			return Workspace{}, fmt.Errorf("invalid Git marker in %s", checkout)
		}
		owner, err := linkedWorktreeOwner(checkout)
		if err != nil {
			return Workspace{}, err
		}
		if owner != "" {
			state = filepath.Join(owner, local, "worktrees", workspaceKey(checkout))
		}
	}
	if workspace != checkout {
		state = filepath.Join(state, "workspaces", workspaceKey(workspace))
	}
	return Workspace{StateRoot: state, LockDir: workspace, git: true}, nil
}

func linkedWorktreeOwner(checkout string) (string, error) {
	marker := filepath.Join(checkout, ".git")
	contents, err := os.ReadFile(marker) //nolint:gosec // G304: discovered Git metadata
	if err != nil {
		return "", fmt.Errorf("read Git worktree marker: %w", err)
	}
	gitDir, ok := strings.CutPrefix(strings.TrimRight(string(contents), "\r\n"), "gitdir: ")
	if !ok || gitDir == "" {
		return "", fmt.Errorf("invalid Git worktree marker %s", marker)
	}
	gitDir, err = resolveGitDirectory(checkout, gitDir)
	if err != nil {
		return "", err
	}
	contents, err = os.ReadFile(filepath.Join(gitDir, "commondir")) //nolint:gosec // G304: discovered Git metadata
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read Git common-directory marker: %w", err)
	}
	commonDir, err := resolveGitDirectory(gitDir, strings.TrimRight(string(contents), "\r\n"))
	if err != nil {
		return "", err
	}
	if filepath.Base(commonDir) != ".git" {
		return commonDir, nil
	}
	bare, err := gitDirectoryIsBare(commonDir)
	if err != nil {
		return "", err
	}
	if !bare {
		return filepath.Dir(commonDir), nil
	}
	return commonDir, nil
}

func gitDirectoryIsBare(gitDir string) (bool, error) {
	path := filepath.Join(gitDir, "config")
	contents, err := os.ReadFile(path) //nolint:gosec // G304: discovered Git metadata
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read Git repository config: %w", err)
	}
	var core, bare bool
	for line := range strings.Lines(string(contents)) {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		line = strings.TrimSpace(strings.SplitN(line, ";", 2)[0])
		if strings.HasPrefix(line, "[") {
			core = strings.EqualFold(line, "[core]")
			continue
		}
		key, value, assigned := strings.Cut(line, "=")
		if !core || !strings.EqualFold(strings.TrimSpace(key), "bare") {
			continue
		}
		bare, err = parseGitBare(value, assigned)
		if err != nil {
			return false, fmt.Errorf("read core.bare in %s: %w", path, err)
		}
	}
	return bare, nil
}

func parseGitBare(value string, assigned bool) (bool, error) {
	if !assigned {
		return true, nil
	}
	switch value := strings.ToLower(strings.Trim(strings.TrimSpace(value), `"`)); value {
	case "true", "yes", "on":
		return true, nil
	case "", "false", "no", "off":
		return false, nil
	default:
		number, err := strconv.Atoi(value)
		if err != nil {
			return false, fmt.Errorf("invalid Git boolean %q", value)
		}
		return number != 0, nil
	}
}

func resolveGitDirectory(base, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty Git directory path under %s", base)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}
	dir, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve Git directory: %w", err)
	}
	info, err := os.Stat(dir) //nolint:gosec // G703: canonical directory from Git metadata
	if err != nil {
		return "", fmt.Errorf("stat Git directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a Git directory: %s", dir)
	}
	return dir, nil
}

func (workspace Workspace) PrepareState() error {
	if err := os.MkdirAll(workspace.StateRoot, 0o700); err != nil {
		return fmt.Errorf("create Circuit state directory: %w", err)
	}
	if !workspace.git {
		return nil
	}
	path := filepath.Join(workspace.StateRoot, ".gitignore")
	const exclude = "*\n"
	content, err := os.ReadFile(path) //nolint:gosec // G304: workspace-private ignore file
	if errors.Is(err, os.ErrNotExist) {
		err = installStateIgnore(path, exclude)
		if !errors.Is(err, os.ErrExist) {
			return err
		}
		content, err = os.ReadFile(path) //nolint:gosec // G304: workspace-private ignore file
	}
	if err != nil {
		return fmt.Errorf("read Circuit state ignore file: %w", err)
	}
	if string(content) != exclude {
		return fmt.Errorf("private state ignore file %s must contain only *; fix its contents or remove it so Circuit can recreate it", path)
	}
	return nil
}

func installStateIgnore(path, contents string) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".circuit-ignore-*")
	if err != nil {
		return fmt.Errorf("create temporary state ignore file: %w", err)
	}
	defer func() {
		_ = os.Remove(file.Name()) //nolint:errcheck // Best-effort temporary file cleanup.
	}()
	_, writeErr := file.WriteString(contents)
	if err := errors.Join(writeErr, file.Sync(), file.Close()); err != nil {
		return fmt.Errorf("write Circuit state ignore file: %w", err)
	}
	if err := os.Link(file.Name(), path); err != nil {
		return fmt.Errorf("install Circuit state ignore file (requires hard links): %w", err)
	}
	return nil
}

func workspaceKey(path string) string {
	sum := sha256.Sum256([]byte(path))
	return hex.EncodeToString(sum[:])
}
