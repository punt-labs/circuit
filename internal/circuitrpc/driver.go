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

type DriverGuidance struct {
	Goal   string
	States map[string]StateGuidance
}

type StateGuidance struct {
	Prompt string
	Event  string
}

func RunUntilAccepted(runtime *circuitrun.Runtime, backend AgentBackend) (DriverResult, error) {
	return RunUntilAcceptedWithGuidance(runtime, backend, DriverGuidance{})
}

func FormatPromptWithGuidance(status circuitrun.StatusReport, guidance DriverGuidance) string {
	prompt := FormatPrompt(status)
	if guidance.Goal != "" {
		prompt += "\nGoal: " + guidance.Goal + "\n"
	}
	stateGuidance, ok := guidance.States[status.Current]
	if !ok {
		return prompt
	}
	if stateGuidance.Prompt != "" {
		prompt += "\nState guidance:\n" + stateGuidance.Prompt + "\n"
	}
	if stateGuidance.Event != "" {
		prompt += "\nWhen ready, request transition event: " + stateGuidance.Event + "\n"
	}
	return prompt
}

func RunUntilAcceptedWithGuidance(runtime *circuitrun.Runtime, backend AgentBackend, guidance DriverGuidance) (DriverResult, error) {
	result := DriverResult{}
	for result.Prompts < maxDriverPrompts {
		status, err := runtime.Status()
		if err != nil {
			return result, err
		}
		if IsTerminal(status) {
			return result, fmt.Errorf("cannot drive terminal state %s", status.Current)
		}
		response, err := backend.Prompt(FormatPromptWithGuidance(status, guidance))
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
