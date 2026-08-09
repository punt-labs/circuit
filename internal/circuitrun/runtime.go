package circuitrun

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/punt-labs/circuit/internal/circuitb"
	"gopkg.in/yaml.v3"
)

type SessionState string

const (
	SessionUnloaded  SessionState = "unloaded"
	SessionActive    SessionState = "active"
	SessionSuspended SessionState = "suspended"
	SessionStopped   SessionState = "stopped"
)

type Runtime struct {
	root      string
	sessions  map[string]*Run
	currentID string
	lastState SessionState
}

type Run struct {
	SessionID   string                  `json:"sessionId"`
	MachineName string                  `json:"machineName"`
	MachineFile string                  `json:"machineFile"`
	Current     string                  `json:"current"`
	Session     SessionState            `json:"session"`
	Booleans    map[string]bool         `json:"booleans,omitempty"`
	Checks      map[string]CheckRuntime `json:"checks,omitempty"`
}

type StatusReport struct {
	SessionID   string
	MachineName string
	Current     string
	Enabled     []circuitb.CallStatus
	Blocked     []circuitb.CallStatus
	Checks      map[string]CheckRuntime
}

type AdvanceReport struct {
	SessionID string
	Allowed   bool
	From      string
	To        string
	Event     string
	Failed    []string
	Checks    map[string]CheckRuntime
}

type CheckRuntime struct {
	Invocations int  `json:"invocations"`
	LastResult  bool `json:"lastResult"`
}

type LoadReport struct {
	MachineName string
	Checks      []CheckBindingReport
}

type CheckBindingReport struct {
	Variable string
	Use      string
	Returns  string
}

type ScaffoldReport struct {
	MachineName          string
	GeneratedBindings    []string
	GeneratedRegistryIDs []string
}

type checkBindingsFile struct {
	Checks map[string]checkBinding `yaml:"checks"`
}

type checkBinding struct {
	Use string `yaml:"use"`
}

type checkRegistryFile struct {
	Checks map[string]registeredCheck `yaml:"checks"`
}

type registeredCheck struct {
	Kind    string `yaml:"kind"`
	Command string `yaml:"command"`
	Returns string `yaml:"returns"`
}

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

func (runtime *Runtime) Suspend() error {
	if err := os.MkdirAll(runtime.sessionsDir(), 0o700); err != nil {
		return err
	}
	for _, id := range runtime.persistedSessionIDs() {
		if _, ok := runtime.sessions[id]; !ok {
			if err := os.Remove(runtime.sessionPath(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	for _, run := range runtime.sessions {
		if run.Session != SessionActive && run.Session != SessionStopped {
			continue
		}
		if err := runtime.writeSession(run); err != nil {
			return err
		}
	}
	return runtime.removeLegacySuspendedRun()
}

func (runtime *Runtime) Load(machineName string) (LoadReport, error) {
	machine, err := circuitb.LoadFile(runtime.resolveMachineFile(machineName))
	if err != nil {
		return LoadReport{}, err
	}
	bindings, registry, err := runtime.loadCheckConfiguration(machineName)
	if err != nil {
		return LoadReport{}, err
	}
	report := LoadReport{MachineName: machineName}
	for _, variable := range machine.BooleanVariables() {
		binding, ok := bindings.Checks[variable]
		if !ok {
			return LoadReport{}, fmt.Errorf("unbound BOOL variable %s; run: circuit scaffold %s", variable, machineName)
		}
		registered, ok := registry.Checks[binding.Use]
		if !ok {
			return LoadReport{}, fmt.Errorf("check %s references unknown registry entry %s", variable, binding.Use)
		}
		if registered.Kind != "command" || registered.Returns != "BOOL" {
			return LoadReport{}, fmt.Errorf("check %s must reference a command returning BOOL", variable)
		}
		report.Checks = append(report.Checks, CheckBindingReport{Variable: variable, Use: binding.Use, Returns: registered.Returns})
	}
	return report, nil
}

func (runtime *Runtime) Scaffold(machineName string) (ScaffoldReport, error) {
	machine, err := circuitb.LoadFile(runtime.resolveMachineFile(machineName))
	if err != nil {
		return ScaffoldReport{}, err
	}
	bindings, err := runtime.loadOptionalCheckBindings(machineName)
	if err != nil {
		return ScaffoldReport{}, err
	}
	registry, err := runtime.loadOptionalCheckRegistry(machineName)
	if err != nil {
		return ScaffoldReport{}, err
	}
	if bindings.Checks == nil {
		bindings.Checks = map[string]checkBinding{}
	}
	if registry.Checks == nil {
		registry.Checks = map[string]registeredCheck{}
	}
	variables := machine.BooleanVariables()
	report := ScaffoldReport{MachineName: machineName}
	if len(variables) == 0 {
		return report, nil
	}
	for _, variable := range variables {
		binding, ok := bindings.Checks[variable]
		if !ok {
			binding = checkBinding{Use: variable}
			bindings.Checks[variable] = binding
			report.GeneratedBindings = append(report.GeneratedBindings, variable)
		}
		if _, ok := registry.Checks[binding.Use]; !ok {
			registry.Checks[binding.Use] = registeredCheck{Kind: "command", Command: "false", Returns: "BOOL"}
			report.GeneratedRegistryIDs = append(report.GeneratedRegistryIDs, binding.Use)
		}
	}
	if err := runtime.writeCheckBindings(machineName, bindings); err != nil {
		return ScaffoldReport{}, err
	}
	if err := runtime.writeCheckRegistry(machineName, registry); err != nil {
		return ScaffoldReport{}, err
	}
	return report, nil
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
	run, err := runtime.singleActiveSession()
	if err != nil {
		return StatusReport{}, err
	}
	return runtime.statusForRun(run)
}

func (runtime *Runtime) StatusAll() ([]StatusReport, error) {
	ids := runtime.activeSessionIDs()
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
	run, err := runtime.activeSessionByID(id)
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
		SessionID:   run.SessionID,
		MachineName: run.MachineName,
		Current:     report.Current,
		Enabled:     report.Enabled,
		Blocked:     report.Blocked,
		Checks:      cloneChecks(run.Checks),
	}
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

func (runtime *Runtime) activeSessionByID(id string) (*Run, error) {
	run, ok := runtime.sessions[id]
	if !ok || run.Session != SessionActive {
		return nil, fmt.Errorf("unknown active session: %s", id)
	}
	return run, nil
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

func (runtime *Runtime) runChecks(run *Run) error {
	bindings, err := runtime.loadCheckBindings(run.MachineName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	registry, err := runtime.loadCheckRegistry(run.MachineName)
	if err != nil {
		return err
	}
	if run.Booleans == nil {
		run.Booleans = map[string]bool{}
	}
	if run.Checks == nil {
		run.Checks = map[string]CheckRuntime{}
	}
	for variable, binding := range bindings.Checks {
		registered, ok := registry.Checks[binding.Use]
		if !ok {
			return fmt.Errorf("check %s references unknown registry entry %s", variable, binding.Use)
		}
		if registered.Kind != "command" || registered.Returns != "BOOL" {
			return fmt.Errorf("check %s must reference a command returning BOOL", variable)
		}
		passed := runtime.runBooleanCommand(registered.Command)
		run.Booleans[variable] = passed
		check := run.Checks[variable]
		check.Invocations++
		check.LastResult = passed
		run.Checks[variable] = check
	}
	return nil
}

func (runtime *Runtime) runBooleanCommand(command string) bool {
	result := exec.Command("sh", "-c", command)
	result.Dir = runtime.root
	return result.Run() == nil
}

func (runtime *Runtime) loadCheckConfiguration(machine string) (checkBindingsFile, checkRegistryFile, error) {
	bindings, err := runtime.loadOptionalCheckBindings(machine)
	if err != nil {
		return checkBindingsFile{}, checkRegistryFile{}, err
	}
	registry, err := runtime.loadOptionalCheckRegistry(machine)
	if err != nil {
		return checkBindingsFile{}, checkRegistryFile{}, err
	}
	if bindings.Checks == nil {
		bindings.Checks = map[string]checkBinding{}
	}
	if registry.Checks == nil {
		registry.Checks = map[string]registeredCheck{}
	}
	return bindings, registry, nil
}

func (runtime *Runtime) loadCheckBindings(machine string) (checkBindingsFile, error) {
	return runtime.readCheckBindings(runtime.checkBindingsPath(machine))
}

func (runtime *Runtime) loadOptionalCheckBindings(machine string) (checkBindingsFile, error) {
	bindings, err := runtime.readCheckBindings(runtime.checkBindingsPath(machine))
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return checkBindingsFile{Checks: map[string]checkBinding{}}, nil
	}
	return bindings, err
}

func (runtime *Runtime) readCheckBindings(path string) (checkBindingsFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return checkBindingsFile{}, err
	}
	var bindings checkBindingsFile
	if err := yaml.Unmarshal(content, &bindings); err != nil {
		return checkBindingsFile{}, err
	}
	return bindings, nil
}

func (runtime *Runtime) writeCheckBindings(machine string, bindings checkBindingsFile) error {
	content, err := yaml.Marshal(bindings)
	if err != nil {
		return err
	}
	return os.WriteFile(runtime.checkBindingsPath(machine), content, 0o600)
}

func (runtime *Runtime) loadCheckRegistry(machine string) (checkRegistryFile, error) {
	return runtime.readCheckRegistry(runtime.checkRegistryPath(machine))
}

func (runtime *Runtime) loadOptionalCheckRegistry(machine string) (checkRegistryFile, error) {
	registry, err := runtime.readCheckRegistry(runtime.checkRegistryPath(machine))
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return checkRegistryFile{Checks: map[string]registeredCheck{}}, nil
	}
	return registry, err
}

func (runtime *Runtime) readCheckRegistry(path string) (checkRegistryFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return checkRegistryFile{}, err
	}
	var registry checkRegistryFile
	if err := yaml.Unmarshal(content, &registry); err != nil {
		return checkRegistryFile{}, err
	}
	return registry, nil
}

func (runtime *Runtime) writeCheckRegistry(machine string, registry checkRegistryFile) error {
	content, err := yaml.Marshal(registry)
	if err != nil {
		return err
	}
	return os.WriteFile(runtime.checkRegistryPath(machine), content, 0o600)
}

func (runtime *Runtime) loadLegacySuspendedRun() error {
	content, err := os.ReadFile(runtime.suspendedPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var run Run
	if err := json.Unmarshal(content, &run); err != nil {
		return err
	}
	if run.Session != SessionActive {
		return nil
	}
	if run.SessionID == "" {
		id, err := runtime.newSessionID(run.MachineName)
		if err != nil {
			return err
		}
		run.SessionID = id
	}
	runtime.sessions[run.SessionID] = &run
	runtime.currentID = run.SessionID
	return nil
}

func (runtime *Runtime) loadSessions() error {
	entries, err := os.ReadDir(runtime.sessionsDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(runtime.sessionsDir(), entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var run Run
		if err := json.Unmarshal(content, &run); err != nil {
			return err
		}
		if run.Session != SessionActive && run.Session != SessionStopped {
			continue
		}
		if run.SessionID == "" {
			run.SessionID = strings.TrimSuffix(entry.Name(), ".json")
		}
		runtime.sessions[run.SessionID] = &run
	}
	return nil
}

func (runtime *Runtime) persistedSessionIDs() []string {
	entries, err := os.ReadDir(runtime.sessionsDir())
	if err != nil {
		return nil
	}
	ids := []string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		ids = append(ids, strings.TrimSuffix(entry.Name(), ".json"))
	}
	sort.Strings(ids)
	return ids
}

func (runtime *Runtime) newSessionID(machineName string) (string, error) {
	base := sessionIDMachineName(machineName)
	for range 16 {
		id, err := randomHex(2)
		if err != nil {
			return "", err
		}
		sessionID := base + "-" + id
		if _, ok := runtime.sessions[sessionID]; ok {
			continue
		}
		if _, err := os.Stat(runtime.sessionPath(sessionID)); errors.Is(err, os.ErrNotExist) {
			return sessionID, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("could not allocate session id for %s", machineName)
}

func sessionIDMachineName(machineName string) string {
	name := strings.TrimSuffix(filepath.Base(machineName), ".mch")
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "machine"
	}
	return name
}

func randomHex(bytesCount int) (string, error) {
	buffer := make([]byte, bytesCount)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func (runtime *Runtime) resolveMachineFile(path string) string {
	if strings.HasSuffix(path, ".mch") || strings.Contains(path, string(filepath.Separator)) {
		return path
	}
	return filepath.Join(runtime.root, "machines", path+".mch")
}

func (runtime *Runtime) checkBindingsPath(machine string) string {
	machineFile := runtime.resolveMachineFile(machine)
	name := strings.TrimSuffix(filepath.Base(machineFile), ".mch")
	return filepath.Join(filepath.Dir(machineFile), name+".checks.yaml")
}

func (runtime *Runtime) checkRegistryPath(machine string) string {
	return filepath.Join(filepath.Dir(runtime.resolveMachineFile(machine)), "check-registry.yaml")
}

func (runtime *Runtime) writeSession(run *Run) error {
	if err := os.MkdirAll(runtime.sessionsDir(), 0o700); err != nil {
		return err
	}
	content, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(runtime.sessionPath(run.SessionID), append(content, '\n'), 0o600)
}

func (runtime *Runtime) sessionPath(id string) string {
	return filepath.Join(runtime.sessionsDir(), id+".json")
}

func (runtime *Runtime) sessionsDir() string {
	return filepath.Join(runtime.root, ".tmp", "sessions")
}

func (runtime *Runtime) suspendedPath() string {
	return filepath.Join(runtime.root, ".tmp", "circuit.suspended.json")
}

func (runtime *Runtime) removeLegacySuspendedRun() error {
	if err := os.Remove(runtime.suspendedPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func cloneChecks(checks map[string]CheckRuntime) map[string]CheckRuntime {
	clone := map[string]CheckRuntime{}
	for key, value := range checks {
		clone[key] = value
	}
	return clone
}
