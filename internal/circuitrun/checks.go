package circuitrun

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

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
	variableNames := make([]string, 0, len(bindings.Checks))
	for name := range bindings.Checks {
		variableNames = append(variableNames, name)
	}
	sort.Strings(variableNames)
	for _, variable := range variableNames {
		binding := bindings.Checks[variable]
		registered, ok := registry.Checks[binding.Use]
		if !ok {
			return fmt.Errorf("check %s references unknown registry entry %s", variable, binding.Use)
		}
		if registered.Kind != checkKindCommand || registered.Returns != checkReturnBool {
			return fmt.Errorf("check %s must reference a command returning BOOL", variable)
		}
		passed := runtime.runBooleanCommand(registered.Command, run)
		run.Booleans[variable] = passed
		check := run.Checks[variable]
		check.Invocations++
		check.LastResult = passed
		run.Checks[variable] = check
	}
	return nil
}

func (runtime *Runtime) runBooleanCommand(command string, run *Run) bool {
	result := exec.Command("sh", "-c", command) //nolint:gosec // G204: command is from check registry controlled by the project owner
	result.Dir = runtime.root
	result.Env = append(os.Environ(),
		"CIRCUIT_SESSION_ID="+run.SessionID,
		"CIRCUIT_MACHINE_NAME="+run.MachineName,
		"CIRCUIT_MACHINE_FILE="+run.MachineFile,
		"CIRCUIT_CURRENT_STATE="+run.Current,
	)
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
	content, err := os.ReadFile(path) //nolint:gosec // G304: path is a project-local check bindings file
	if err != nil {
		return checkBindingsFile{}, fmt.Errorf("read check bindings %s: %w", path, err)
	}
	var bindings checkBindingsFile
	if err := yaml.Unmarshal(content, &bindings); err != nil {
		return checkBindingsFile{}, fmt.Errorf("parse check bindings %s: %w", path, err)
	}
	return bindings, nil
}

func (runtime *Runtime) writeCheckBindings(machine string, bindings checkBindingsFile) error {
	content, err := yaml.Marshal(bindings)
	if err != nil {
		return fmt.Errorf("marshal check bindings: %w", err)
	}
	if err := os.WriteFile(runtime.checkBindingsPath(machine), content, 0o600); err != nil {
		return fmt.Errorf("write check bindings: %w", err)
	}
	return nil
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
	content, err := os.ReadFile(path) //nolint:gosec // G304: path is a project-local check registry file
	if err != nil {
		return checkRegistryFile{}, fmt.Errorf("read check registry %s: %w", path, err)
	}
	var registry checkRegistryFile
	if err := yaml.Unmarshal(content, &registry); err != nil {
		return checkRegistryFile{}, fmt.Errorf("parse check registry %s: %w", path, err)
	}
	return registry, nil
}

func (runtime *Runtime) writeCheckRegistry(machine string, registry checkRegistryFile) error {
	content, err := yaml.Marshal(registry)
	if err != nil {
		return fmt.Errorf("marshal check registry: %w", err)
	}
	if err := os.WriteFile(runtime.checkRegistryPath(machine), content, 0o600); err != nil {
		return fmt.Errorf("write check registry: %w", err)
	}
	return nil
}

func (runtime *Runtime) checkBindingsPath(machine string) string {
	machineFile := runtime.resolveMachineFile(machine)
	name := strings.TrimSuffix(filepath.Base(machineFile), ".mch")
	return filepath.Join(filepath.Dir(machineFile), name+".checks.yaml")
}

func (runtime *Runtime) checkRegistryPath(machine string) string {
	return filepath.Join(filepath.Dir(runtime.resolveMachineFile(machine)), "check-registry.yaml")
}
