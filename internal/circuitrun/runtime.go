package circuitrun

import (
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

type Runtime struct {
	root string
	run  *Run
}

type Run struct {
	MachineName string                  `json:"machineName"`
	MachineFile string                  `json:"machineFile"`
	Current     string                  `json:"current"`
	Booleans    map[string]bool         `json:"booleans,omitempty"`
	Checks      map[string]CheckRuntime `json:"checks,omitempty"`
}

type CheckRuntime struct {
	Invocations int  `json:"invocations"`
	LastResult  bool `json:"lastResult"`
}

type StatusReport struct {
	MachineName string
	Current     string
	Enabled     []circuitb.CallStatus
	Blocked     []circuitb.CallStatus
	Checks      map[string]CheckRuntime
}

type AdvanceReport struct {
	Allowed bool
	From    string
	To      string
	Event   string
	Failed  []string
	Checks  map[string]CheckRuntime
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

func Resume(root string) (*Runtime, error) {
	runtime := &Runtime{root: root}
	content, err := os.ReadFile(runtime.suspendedPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return runtime, nil
		}
		return nil, err
	}
	var run Run
	if err := json.Unmarshal(content, &run); err != nil {
		return nil, err
	}
	runtime.run = &run
	return runtime, nil
}

func (runtime *Runtime) Suspend() error {
	if runtime.run == nil {
		return nil
	}
	path := runtime.suspendedPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	content, err := json.MarshalIndent(runtime.run, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(content, '\n'), 0o600)
}

func (runtime *Runtime) Start(machineName string) (StatusReport, error) {
	machineFile := runtime.resolveMachineFile(machineName)
	machine, err := circuitb.LoadFile(machineFile)
	if err != nil {
		return StatusReport{}, err
	}
	report, err := machine.State(nil)
	if err != nil {
		return StatusReport{}, err
	}
	runtime.run = &Run{
		MachineName: machineName,
		MachineFile: machineFile,
		Current:     report.Current,
		Booleans:    map[string]bool{},
		Checks:      map[string]CheckRuntime{},
	}
	return runtime.statusFromReport(report), nil
}

func (runtime *Runtime) Status() (StatusReport, error) {
	if runtime.run == nil {
		return StatusReport{}, errors.New("no suspended circuit; run: circuit start <machine>")
	}
	machine, err := circuitb.LoadFile(runtime.run.MachineFile)
	if err != nil {
		return StatusReport{}, err
	}
	report, err := machine.StateAtWithBooleans(runtime.run.Current, runtime.run.Booleans)
	if err != nil {
		return StatusReport{}, err
	}
	return runtime.statusFromReport(report), nil
}

func (runtime *Runtime) Advance(event string) (AdvanceReport, error) {
	if runtime.run == nil {
		return AdvanceReport{}, errors.New("no suspended circuit; run: circuit start <machine>")
	}
	machine, err := circuitb.LoadFile(runtime.run.MachineFile)
	if err != nil {
		return AdvanceReport{}, err
	}
	if err := runtime.runChecks(); err != nil {
		return AdvanceReport{}, err
	}
	result, err := machine.AdvanceFromWithBooleans(event, runtime.run.Current, runtime.run.Booleans)
	if err != nil {
		return AdvanceReport{}, err
	}
	report := AdvanceReport{
		Allowed: result.Allowed,
		From:    result.From,
		To:      result.To,
		Event:   event,
		Failed:  result.Failed,
		Checks:  cloneChecks(runtime.run.Checks),
	}
	if result.Allowed {
		runtime.run.Current = result.To
	}
	return report, nil
}

func (runtime *Runtime) Stop() error {
	runtime.run = nil
	if err := os.Remove(runtime.suspendedPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
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

func (runtime *Runtime) statusFromReport(report circuitb.StateReport) StatusReport {
	return StatusReport{
		MachineName: runtime.run.MachineName,
		Current:     report.Current,
		Enabled:     report.Enabled,
		Blocked:     report.Blocked,
		Checks:      cloneChecks(runtime.run.Checks),
	}
}

func (runtime *Runtime) runChecks() error {
	bindings, err := runtime.loadCheckBindings(runtime.run.MachineName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	registry, err := runtime.loadCheckRegistry()
	if err != nil {
		return err
	}
	if runtime.run.Booleans == nil {
		runtime.run.Booleans = map[string]bool{}
	}
	if runtime.run.Checks == nil {
		runtime.run.Checks = map[string]CheckRuntime{}
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
		runtime.run.Booleans[variable] = passed
		check := runtime.run.Checks[variable]
		check.Invocations++
		check.LastResult = passed
		runtime.run.Checks[variable] = check
	}
	return nil
}

func (runtime *Runtime) runBooleanCommand(command string) bool {
	result := exec.Command("sh", "-c", command)
	result.Dir = runtime.root
	return result.Run() == nil
}

func (runtime *Runtime) loadCheckBindings(machine string) (checkBindingsFile, error) {
	path := filepath.Join(runtime.root, "machines", machine+".checks.yaml")
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

func (runtime *Runtime) loadCheckRegistry() (checkRegistryFile, error) {
	path := filepath.Join(runtime.root, "machines", "check-registry.yaml")
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

func (runtime *Runtime) resolveMachineFile(path string) string {
	if strings.HasSuffix(path, ".mch") || strings.Contains(path, string(filepath.Separator)) {
		return path
	}
	return filepath.Join(runtime.root, "machines", path+".mch")
}

func (runtime *Runtime) suspendedPath() string {
	return filepath.Join(runtime.root, ".tmp", "circuit.suspended.json")
}

func cloneChecks(checks map[string]CheckRuntime) map[string]CheckRuntime {
	clone := map[string]CheckRuntime{}
	for key, value := range checks {
		clone[key] = value
	}
	return clone
}
