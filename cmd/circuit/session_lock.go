package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/punt-labs/circuit/internal/circuitpath"
)

func (cmd command) dispatchWithSessionLock(call invocation) (bool, error) {
	var mode int
	switch call.action {
	case actionStatus:
		mode = syscall.LOCK_SH
	case actionSessionList, actionSessionShow:
		mode = syscall.LOCK_EX
		if cmd.json {
			mode = syscall.LOCK_SH
		}
	case actionScaffold, actionStart, actionStop, actionRemove:
		mode = syscall.LOCK_EX
	default:
		return cmd.dispatch(call)
	}
	lock, err := cmd.lockSessions(mode)
	if err != nil {
		return false, err
	}
	defer cmd.closeSessionLock(lock)
	return cmd.dispatch(call)
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
