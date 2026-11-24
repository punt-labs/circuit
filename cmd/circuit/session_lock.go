package main

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"syscall"

	"github.com/punt-labs/circuit/internal/circuitpath"
)

func (cmd command) dispatchWithSessionLock(action actionName, args []string) (bool, error) {
	var mode int
	switch action {
	case actionStatus:
		mode = syscall.LOCK_EX
		if slices.Contains(args[1:], jsonFlag) {
			mode = syscall.LOCK_SH
		}
	case actionScaffold, actionStart, actionStop, actionUnload:
		mode = syscall.LOCK_EX
	default:
		return cmd.dispatch(action, args)
	}
	lock, err := cmd.lockSessions(mode)
	if err != nil {
		return false, err
	}
	defer cmd.closeSessionLock(lock)
	return cmd.dispatch(action, args)
}

func (cmd command) lockSessions(mode int) (*os.File, error) {
	workspace, err := circuitpath.ResolveWorkspace(cmd.workingDir())
	if err != nil {
		return nil, fmt.Errorf("resolve session workspace: %w", err)
	}
	lock, err := os.Open(workspace.LockDir) //nolint:gosec // G304: lock the workspace directory itself
	if err != nil {
		return nil, fmt.Errorf("open session lock: %w", err)
	}
	for {
		err = syscall.Flock(int(lock.Fd()), mode)
		if !errors.Is(err, syscall.EINTR) {
			break
		}
	}
	if err != nil {
		cmd.closeSessionLock(lock)
		return nil, fmt.Errorf("lock sessions: %w", err)
	}
	return lock, nil
}

func (cmd command) closeSessionLock(lock *os.File) {
	if err := lock.Close(); err != nil {
		cmd.diagnostics().Warn("Could not close the session lock")
	}
}
