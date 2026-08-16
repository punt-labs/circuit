package circuitrun

import (
	"github.com/punt-labs/circuit/internal/circuitb"
)

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
			registry.Checks[binding.Use] = registeredCheck{Kind: checkKindCommand, Command: "false", Returns: checkReturnBool}
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
