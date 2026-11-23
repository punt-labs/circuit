package circuitrun

import (
	"fmt"
	"maps"
	"reflect"
)

type AdvanceUpdate struct {
	runtime *Runtime
	run     *Run
	before  Run
	event   string
}

func (runtime *Runtime) PrepareAdvance(id, event string) (*AdvanceUpdate, error) {
	run, err := runtime.selectAdvanceSession(id)
	if err != nil {
		return nil, err
	}
	if runtime.legacyNeedsID {
		if err := runtime.Suspend(); err != nil {
			return nil, err
		}
	}
	before := *run
	before.Booleans = maps.Clone(run.Booleans)
	before.Checks = maps.Clone(run.Checks)
	return &AdvanceUpdate{runtime: runtime, run: run, before: before, event: event}, nil
}

func (runtime *Runtime) selectAdvanceSession(id string) (*Run, error) {
	if id != "" {
		return runtime.activeSessionByID(id)
	}
	return runtime.singleActiveSession()
}

func (update *AdvanceUpdate) Execute() (AdvanceReport, error) {
	return update.runtime.advanceRun(update.run, update.event)
}

func (update *AdvanceUpdate) Commit(current *Runtime) error {
	run, ok := current.sessions[update.before.SessionID]
	if !ok || !sameRun(&update.before, run) {
		return fmt.Errorf("session %s changed during advance; retry the command", update.before.SessionID)
	}
	current.sessions[update.before.SessionID] = update.run
	return current.Suspend()
}

func sameRun(a, b *Run) bool {
	if !maps.Equal(a.Booleans, b.Booleans) || !maps.Equal(a.Checks, b.Checks) {
		return false
	}
	aFields, bFields := *a, *b
	aFields.Booleans, bFields.Booleans = nil, nil
	aFields.Checks, bFields.Checks = nil, nil
	return reflect.DeepEqual(aFields, bFields)
}
