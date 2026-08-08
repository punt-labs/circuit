package playbook

import (
	"fmt"
	"strings"
)

type Diagnostic struct {
	Path    string
	Message string
}

func (diagnostic Diagnostic) String() string {
	if diagnostic.Path == "" {
		return diagnostic.Message
	}
	return diagnostic.Path + ": " + diagnostic.Message
}

type ValidationResult struct {
	Diagnostics []Diagnostic
}

func (result ValidationResult) OK() bool {
	return len(result.Diagnostics) == 0
}

func (result ValidationResult) Error() string {
	messages := make([]string, 0, len(result.Diagnostics))
	for _, diagnostic := range result.Diagnostics {
		messages = append(messages, diagnostic.String())
	}
	return strings.Join(messages, "\n")
}

func Validate(playbook Playbook) ValidationResult {
	validator := validator{}
	validator.validate(playbook)
	return ValidationResult{Diagnostics: validator.diagnostics}
}

type validator struct {
	diagnostics []Diagnostic
}

func (validator *validator) validate(playbook Playbook) {
	if strings.TrimSpace(playbook.Name) == "" {
		validator.add("name", "required")
	}

	if playbook.IsLinear() == playbook.IsMachine() {
		validator.add("", "exactly one of steps or states is required")
	}

	if playbook.IsMachine() {
		validator.validateMachine(playbook.States)
	}
}

func (validator *validator) validateMachine(states []State) {
	ids := make(map[string]int, len(states))
	for index, state := range states {
		path := fmt.Sprintf("states[%d]", index)
		id := strings.TrimSpace(state.ID)
		if id == "" {
			validator.add(path+".id", "required")
			continue
		}
		if previous, ok := ids[id]; ok {
			validator.add(path+".id", fmt.Sprintf("duplicate state id %q also used at states[%d]", id, previous))
			continue
		}
		ids[id] = index
	}

	for index, state := range states {
		validator.validateState(index, state, ids)
	}
}

func (validator *validator) validateState(index int, state State, ids map[string]int) {
	path := fmt.Sprintf("states[%d]", index)
	if strings.TrimSpace(state.Description) == "" {
		validator.add(path+".description", "required")
	}

	if len(state.Transitions) == 0 {
		return
	}

	for transitionIndex, transition := range state.Transitions {
		transitionPath := fmt.Sprintf("%s.transitions[%d]", path, transitionIndex)
		if strings.TrimSpace(transition.To) == "" {
			validator.add(transitionPath+".to", "required")
		} else if _, ok := ids[transition.To]; !ok {
			validator.add(transitionPath+".to", fmt.Sprintf("unknown target state %q", transition.To))
		}
	}

	if state.Poll == "" && !hasUnconditionalTransition(state.Transitions) {
		validator.add(path, "non-terminal state has guarded transitions but no poll or unconditional fallback")
	}
}

func hasUnconditionalTransition(transitions []Transition) bool {
	for _, transition := range transitions {
		if len(transition.When) == 0 {
			return true
		}
	}
	return false
}

func (validator *validator) add(path string, message string) {
	validator.diagnostics = append(validator.diagnostics, Diagnostic{Path: path, Message: message})
}
