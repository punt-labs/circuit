package circuitb

import "sort"

func (machine Machine) BooleanVariables() []string {
	variables := []string{}
	for name, variable := range machine.variables {
		if variable.kind == valueBool {
			variables = append(variables, name)
		}
	}
	sort.Strings(variables)
	return variables
}
