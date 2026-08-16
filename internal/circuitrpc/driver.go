package circuitrpc

import (
	"fmt"

	"github.com/punt-labs/circuit/internal/circuitrun"
)

const maxDriverPrompts = 10

type AgentBackend interface {
	Prompt(message string) (string, error)
}

type DriverResult struct {
	Prompts    int
	Transition circuitrun.AdvanceReport
}

func RunUntilAccepted(runtime *circuitrun.Runtime, backend AgentBackend) (DriverResult, error) {
	result := DriverResult{}
	for result.Prompts < maxDriverPrompts {
		status, err := runtime.Status()
		if err != nil {
			return result, err
		}
		if IsTerminal(status) {
			return result, fmt.Errorf("cannot drive terminal state %s", status.Current)
		}
		response, err := backend.Prompt(FormatPrompt(status))
		if err != nil {
			return result, err
		}
		result.Prompts++
		operation := ExtractOperation(response, status)
		if operation == "" {
			continue
		}
		report, err := runtime.Advance(operation)
		if err != nil {
			return result, err
		}
		if !report.Allowed {
			continue
		}
		result.Transition = report
		return result, nil
	}
	return result, fmt.Errorf("no accepted operation after %d prompts", result.Prompts)
}
