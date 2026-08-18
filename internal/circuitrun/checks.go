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

func loadOptionalYAMLFile[T any](path, label string, empty T) (T, error) {
	result, err := readYAMLFile[T](path, label)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return empty, nil
	}
	return result, err
}

func (runtime *Runtime) loadOptionalCheckBindings(machine string) (checkBindingsFile, error) {
	return loadOptionalYAMLFile(runtime.checkBindingsPath(machine), "check bindings",
		checkBindingsFile{Checks: map[string]checkBinding{}})
}

func readYAMLFile[T any](path, label string) (T, error) {
	var zero T
	content, err := os.ReadFile(path) //nolint:gosec // G304: path is a project-local file resolved by runtime
	if err != nil {
		return zero, fmt.Errorf("read %s %s: %w", label, path, err)
	}
	var result T
	if err := yaml.Unmarshal(content, &result); err != nil {
		return zero, fmt.Errorf("parse %s %s: %w", label, path, err)
	}
	return result, nil
}

func writeYAMLFile(path, label string, value any) error {
	content, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", label, err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil { //nolint:gosec // G306: 0o600 is intentionally restrictive
		return fmt.Errorf("write %s %s: %w", label, path, err)
	}
	return nil
}

func (runtime *Runtime) readCheckBindings(path string) (checkBindingsFile, error) {
	return readYAMLFile[checkBindingsFile](path, "check bindings")
}

func (runtime *Runtime) writeCheckBindings(machine string, bindings checkBindingsFile) error {
	return writeYAMLFile(runtime.checkBindingsPath(machine), "check bindings", bindings)
}

func (runtime *Runtime) loadCheckRegistry(machine string) (checkRegistryFile, error) {
	return runtime.readCheckRegistry(runtime.checkRegistryPath(machine))
}

func (runtime *Runtime) loadOptionalCheckRegistry(machine string) (checkRegistryFile, error) {
	return loadOptionalYAMLFile(runtime.checkRegistryPath(machine), "check registry",
		checkRegistryFile{Checks: map[string]registeredCheck{}})
}

func (runtime *Runtime) readCheckRegistry(path string) (checkRegistryFile, error) {
	return readYAMLFile[checkRegistryFile](path, "check registry")
}

func (runtime *Runtime) writeCheckRegistry(machine string, registry checkRegistryFile) error {
	return writeYAMLFile(runtime.checkRegistryPath(machine), "check registry", registry)
}

func (runtime *Runtime) checkBindingsPath(machine string) string {
	machineFile := runtime.resolveMachineFile(machine)
	name := strings.TrimSuffix(filepath.Base(machineFile), ".mch")
	return filepath.Join(filepath.Dir(machineFile), name+".checks.yaml")
}

func (runtime *Runtime) checkRegistryPath(machine string) string {
	return filepath.Join(filepath.Dir(runtime.resolveMachineFile(machine)), "check-registry.yaml")
}
