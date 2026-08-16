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

type SessionDriverResult struct {
	Prompts     int
	Transitions []circuitrun.AdvanceReport
}

type DriverGuidance struct {
	Goal   string
	States map[string]StateGuidance
}

type StateGuidance struct {
	Prompt string
	Event  string
}

func RunGuidedSession(runtime *circuitrun.Runtime, backend AgentBackend, guidance DriverGuidance) (SessionDriverResult, error) {
	result := SessionDriverResult{}
	for runtime.IsActive() {
		step, err := RunUntilAcceptedWithGuidance(runtime, backend, guidance)
		result.Prompts += step.Prompts
		if err != nil {
			return result, err
		}
		result.Transitions = append(result.Transitions, step.Transition)
	}
	return result, nil
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
		if status.SessionState == circuitrun.SessionStopped {
			return result, fmt.Errorf("cannot drive stopped state %s", status.Current)
		}
		response, err := backend.Prompt(FormatPromptWithGuidance(status, guidance))
		if err != nil {
			return result, err
		}
		result.Prompts++
		operation := ExtractRequestedOperation(response, status)
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
