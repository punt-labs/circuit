package circuitrun

import (
	"fmt"
	"github.com/punt-labs/circuit/internal/circuitb"
)

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
