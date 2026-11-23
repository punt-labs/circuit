package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func (cmd command) dispatchWithSessionLock(action actionName, args []string) (bool, error) {
	var mode int
	switch action {
	case actionList, actionLoad, actionScaffold:
		mode = syscall.LOCK_SH
	case actionStart, actionStatus, actionAdvance, actionStop, actionUnload:
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
	dir := filepath.Join(cmd.workingDir(), ".tmp")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create session lock directory: %w", err)
	}
	lock, err := os.OpenFile(filepath.Join(dir, "sessions.lock"), os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // G304: fixed lock path under the workspace
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
