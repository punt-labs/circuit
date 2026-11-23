package main

import (
	"os"
	"syscall"

	"github.com/punt-labs/circuit/internal/circuitrun"
)

func (cmd command) prepareAdvance(session, event string) (*circuitrun.AdvanceUpdate, error) {
	runtime, lock, err := cmd.resumeForAdvance()
	if err != nil {
		return nil, err
	}
	defer cmd.closeSessionLock(lock)
	return runtime.PrepareAdvance(session, event)
}

func (cmd command) commitAdvance(pending *circuitrun.AdvanceUpdate) error {
	runtime, lock, err := cmd.resumeForAdvance()
	if err != nil {
		return err
	}
	defer cmd.closeSessionLock(lock)
	return pending.Commit(runtime)
}

func (cmd command) resumeForAdvance() (*circuitrun.Runtime, *os.File, error) {
	lock, err := cmd.lockSessions(syscall.LOCK_EX)
	if err != nil {
		return nil, nil, err
	}
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		cmd.closeSessionLock(lock)
		return nil, nil, err
	}
	return runtime, lock, nil
}
