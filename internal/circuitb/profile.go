package circuitb

func validateProfile(raw rawMachine) error {
	validator := profileValidator{}
	validator.machine(raw)
	return validator.diagnostics.Err()
}

type profileValidator struct {
	diagnostics Diagnostics
}

func (validator *profileValidator) machine(raw rawMachine) {
	if raw.Name == "" {
		validator.diagnostics = append(validator.diagnostics, Diagnostic{Span: raw.Span, Message: "machine name is required"})
	}
	validator.substitution(raw.Initialisation)
	for _, operation := range raw.Operations {
		validator.operation(operation)
	}
}

func (validator *profileValidator) operation(operation rawOperation) {
	validator.substitution(operation.Body)
}

func (validator *profileValidator) substitution(substitution rawSubstitution) {
	switch item := substitution.(type) {
	case rawUnsupportedSubstitution:
		validator.diagnostics = append(validator.diagnostics, Diagnostic{Span: item.Span, Message: "unsupported B construct: " + item.Construct, Hint: "Circuit-B currently supports assignment, parallel assignment, IF/ELSIF, and PRE/THEN"})
		return
	case rawAssignment:
		return
	case rawParallelAssignment:
		return
	case rawIfSubstitution:
		for _, branch := range item.Branches {
			validator.substitution(branch.Body)
		}
		if item.Else != nil {
			validator.substitution(item.Else)
		}
	default:
		validator.diagnostics = append(validator.diagnostics, Diagnostic{Span: substitution.substitutionSpan(), Message: "unsupported B construct", Hint: "Circuit-B currently supports assignment, parallel assignment, IF/ELSIF, and PRE/THEN"})
	}
}
