package circuitrun

import (
	"errors"
	"fmt"
	"github.com/punt-labs/circuit/internal/circuitb"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (runtime *Runtime) IsActive() bool {
	return len(runtime.activeSessionIDs()) > 0
}

func (runtime *Runtime) SessionState() SessionState {
	ids := runtime.activeSessionIDs()
	if len(ids) > 0 {
		return SessionActive
	}
	if runtime.lastState != "" {
		return runtime.lastState
	}
	return SessionUnloaded
}

func Resume(root string) (*Runtime, error) {
	runtime := &Runtime{root: root, sessions: map[string]*Run{}, lastState: SessionUnloaded}
	if err := runtime.loadLegacySuspendedRun(); err != nil {
		return nil, err
	}
	if err := runtime.loadSessions(); err != nil {
		return nil, err
	}
	if len(runtime.activeSessionIDs()) > 0 {
		runtime.lastState = SessionActive
	} else if len(runtime.stoppedSessionIDs()) > 0 {
		runtime.lastState = SessionStopped
	}
	return runtime, nil
}

func (runtime *Runtime) Start(machineName string) (string, StatusReport, error) {
	if _, err := runtime.Load(machineName); err != nil {
		return "", StatusReport{}, err
	}
	machineFile := runtime.resolveMachineFile(machineName)
	machine, err := circuitb.LoadFile(machineFile)
	if err != nil {
		return "", StatusReport{}, err
	}
	report, err := machine.State(nil)
	if err != nil {
		return "", StatusReport{}, err
	}
	id, err := runtime.newSessionID(machineName)
	if err != nil {
		return "", StatusReport{}, err
	}
	run := &Run{
		SessionID:   id,
		MachineName: machineName,
		MachineFile: machineFile,
		Current:     report.Current,
		Session:     SessionActive,
		Booleans:    map[string]bool{},
		Checks:      map[string]CheckRuntime{},
	}
	runtime.sessions[id] = run
	runtime.currentID = id
	runtime.lastState = SessionActive
	return id, runtime.statusFromReport(run, report), nil
}

func (runtime *Runtime) Status() (StatusReport, error) {
	run, err := runtime.singleKnownSession()
	if err != nil {
		return StatusReport{}, err
	}
	return runtime.statusForRun(run)
}

func (runtime *Runtime) StatusAll() ([]StatusReport, error) {
	ids := runtime.knownSessionIDs()
	reports := make([]StatusReport, 0, len(ids))
	for _, id := range ids {
		report, err := runtime.StatusByID(id)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func (runtime *Runtime) StatusByID(id string) (StatusReport, error) {
	run, err := runtime.knownSessionByID(id)
	if err != nil {
		return StatusReport{}, err
	}
	return runtime.statusForRun(run)
}

func (runtime *Runtime) Advance(event string) (AdvanceReport, error) {
	run, err := runtime.singleActiveSession()
	if err != nil {
		return AdvanceReport{}, err
	}
	return runtime.advanceRun(run, event)
}

func (runtime *Runtime) AdvanceByID(id string, event string) (AdvanceReport, error) {
	run, err := runtime.activeSessionByID(id)
	if err != nil {
		return AdvanceReport{}, err
	}
	return runtime.advanceRun(run, event)
}

func (runtime *Runtime) Stop() error {
	activeIDs := runtime.activeSessionIDs()
	if len(activeIDs) > 1 {
		return fmt.Errorf("multiple active sessions; specify one of: %s", strings.Join(activeIDs, ", "))
	}
	if len(activeIDs) == 1 {
		return runtime.StopByID(activeIDs[0])
	}
	stoppedIDs := runtime.stoppedSessionIDs()
	if len(stoppedIDs) > 1 {
		return fmt.Errorf("multiple stopped sessions; specify one of: %s", strings.Join(stoppedIDs, ", "))
	}
	if len(stoppedIDs) == 1 {
		return nil
	}
	return errors.New("no session to stop; run: circuit start <machine>")
}

func (runtime *Runtime) StopByID(id string) error {
	run, ok := runtime.sessions[id]
	if !ok {
		return fmt.Errorf("unknown session: %s", id)
	}
	if run.Session != SessionStopped {
		run.Session = SessionStopped
	}
	if runtime.currentID == id {
		runtime.currentID = ""
	}
	if len(runtime.activeSessionIDs()) == 0 {
		runtime.lastState = SessionStopped
	}
	if err := runtime.writeSession(run); err != nil {
		return err
	}
	return runtime.removeLegacySuspendedRun()
}

func (runtime *Runtime) UnloadByID(id string) error {
	run, ok := runtime.sessions[id]
	if !ok {
		return fmt.Errorf("unknown session: %s", id)
	}
	if run.Session == SessionActive {
		return fmt.Errorf("cannot unload active session: %s", id)
	}
	delete(runtime.sessions, id)
	if runtime.currentID == id {
		runtime.currentID = ""
	}
	if err := os.Remove(runtime.sessionPath(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if len(runtime.knownSessionIDs()) == 0 {
		runtime.lastState = SessionUnloaded
	}
	return runtime.removeLegacySuspendedRun()
}

func (runtime *Runtime) ListMachines() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(runtime.root, "machines"))
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".mch") {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".mch"))
	}
	sort.Strings(names)
	return names, nil
}

func (runtime *Runtime) SuspendedPath() string {
	return runtime.suspendedPath()
}

func (runtime *Runtime) advanceRun(run *Run, event string) (AdvanceReport, error) {
	machine, err := circuitb.LoadFile(run.MachineFile)
	if err != nil {
		return AdvanceReport{}, err
	}
	if err := runtime.runChecks(run); err != nil {
		return AdvanceReport{}, err
	}
	result, err := machine.AdvanceFromWithBooleans(event, run.Current, run.Booleans)
	if err != nil {
		return AdvanceReport{}, err
	}
	report := AdvanceReport{
		SessionID: run.SessionID,
		Allowed:   result.Allowed,
		From:      result.From,
		To:        result.To,
		Event:     event,
		Failed:    result.Failed,
		Checks:    cloneChecks(run.Checks),
	}
	if result.Allowed {
		run.Current = result.To
		runtime.currentID = run.SessionID
		if runtime.isTerminal(run) {
			run.Session = SessionStopped
			runtime.lastState = SessionStopped
		}
	}
	return report, nil
}

func (runtime *Runtime) statusForRun(run *Run) (StatusReport, error) {
	machine, err := circuitb.LoadFile(run.MachineFile)
	if err != nil {
		return StatusReport{}, err
	}
	report, err := machine.StateAtWithBooleans(run.Current, run.Booleans)
	if err != nil {
		return StatusReport{}, err
	}
	return runtime.statusFromReport(run, report), nil
}

func (runtime *Runtime) statusFromReport(run *Run, report circuitb.StateReport) StatusReport {
	return StatusReport{
		SessionID:    run.SessionID,
		SessionState: run.Session,
		MachineName:  run.MachineName,
		Current:      report.Current,
		Enabled:      report.Enabled,
		Blocked:      report.Blocked,
		Checks:       cloneChecks(run.Checks),
	}
}

func (runtime *Runtime) singleKnownSession() (*Run, error) {
	ids := runtime.knownSessionIDs()
	if len(ids) == 0 {
		return nil, errors.New("no session; run: circuit start <machine>")
	}
	if len(ids) > 1 {
		return nil, fmt.Errorf("multiple sessions; specify one of: %s", strings.Join(ids, ", "))
	}
	return runtime.sessions[ids[0]], nil
}

func (runtime *Runtime) singleActiveSession() (*Run, error) {
	ids := runtime.activeSessionIDs()
	if len(ids) == 0 {
		return nil, errors.New("no active session; run: circuit start <machine>")
	}
	if len(ids) > 1 {
		return nil, fmt.Errorf("multiple active sessions; specify one of: %s", strings.Join(ids, ", "))
	}
	return runtime.sessions[ids[0]], nil
}

func (runtime *Runtime) knownSessionByID(id string) (*Run, error) {
	run, ok := runtime.sessions[id]
	if !ok {
		return nil, fmt.Errorf("unknown session: %s", id)
	}
	return run, nil
}

func (runtime *Runtime) activeSessionByID(id string) (*Run, error) {
	run, err := runtime.knownSessionByID(id)
	if err != nil || run.Session != SessionActive {
		return nil, fmt.Errorf("unknown active session: %s", id)
	}
	return run, nil
}

func (runtime *Runtime) knownSessionIDs() []string {
	ids := make([]string, 0, len(runtime.sessions))
	for id, run := range runtime.sessions {
		if run.Session == SessionActive || run.Session == SessionStopped {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func (runtime *Runtime) activeSessionIDs() []string {
	return runtime.sessionIDsByState(SessionActive)
}

func (runtime *Runtime) stoppedSessionIDs() []string {
	return runtime.sessionIDsByState(SessionStopped)
}

func (runtime *Runtime) sessionIDsByState(state SessionState) []string {
	ids := make([]string, 0, len(runtime.sessions))
	for id, run := range runtime.sessions {
		if run.Session == state {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func (runtime *Runtime) isTerminal(run *Run) bool {
	machine, err := circuitb.LoadFile(run.MachineFile)
	if err != nil {
		return false
	}
	report, err := machine.StateAtWithBooleans(run.Current, run.Booleans)
	if err != nil {
		return false
	}
	return len(report.Enabled) == 0
}

func (runtime *Runtime) resolveMachineFile(path string) string {
	if strings.HasSuffix(path, ".mch") || strings.Contains(path, string(filepath.Separator)) {
		return path
	}
	return filepath.Join(runtime.root, "machines", path+".mch")
}

func cloneChecks(checks map[string]CheckRuntime) map[string]CheckRuntime {
	clone := map[string]CheckRuntime{}
	for key, value := range checks {
		clone[key] = value
	}
	return clone
}
