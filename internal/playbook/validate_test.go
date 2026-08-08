package playbook

import (
	"strings"
	"testing"
)

func TestValidateLinearPlaybook(t *testing.T) {
	t.Parallel()
	result := Validate(Playbook{Name: "linear", Steps: []Action{{ID: "one"}}})

	if !result.OK() {
		t.Fatalf("linear playbook failed validation: %s", result.Error())
	}
}

func TestValidateMachinePlaybook(t *testing.T) {
	t.Parallel()
	result := Validate(Playbook{
		Name: "machine",
		States: []State{
			{ID: "watch", Description: "Watch", Poll: "5m", Transitions: []Transition{{To: "done", When: []GuardCheck{{Description: "Ready", Check: "ready"}}}}},
			{ID: "done", Description: "Done"},
		},
	})

	if !result.OK() {
		t.Fatalf("machine playbook failed validation: %s", result.Error())
	}
}

func TestValidateRequiresExactlyOneShape(t *testing.T) {
	t.Parallel()
	result := Validate(Playbook{Name: "bad", Steps: []Action{{ID: "one"}}, States: []State{{ID: "state", Description: "State"}}})

	assertDiagnosticContains(t, result, "exactly one")
}

func TestValidateRejectsDuplicateStateIDs(t *testing.T) {
	t.Parallel()
	result := Validate(Playbook{
		Name: "dupe",
		States: []State{
			{ID: "same", Description: "One"},
			{ID: "same", Description: "Two"},
		},
	})

	assertDiagnosticContains(t, result, "duplicate state id")
}

func TestValidateRejectsUnknownTransitionTarget(t *testing.T) {
	t.Parallel()
	result := Validate(Playbook{
		Name: "unknown",
		States: []State{
			{ID: "watch", Description: "Watch", Poll: "5m", Transitions: []Transition{{To: "missing"}}},
		},
	})

	assertDiagnosticContains(t, result, "unknown target state")
}

func TestValidateRejectsStuckGuardedState(t *testing.T) {
	t.Parallel()
	result := Validate(Playbook{
		Name: "stuck",
		States: []State{
			{ID: "watch", Description: "Watch", Transitions: []Transition{{To: "done", When: []GuardCheck{{Description: "Ready", Check: "ready"}}}}},
			{ID: "done", Description: "Done"},
		},
	})

	assertDiagnosticContains(t, result, "non-terminal state")
}

func assertDiagnosticContains(t *testing.T, result ValidationResult, text string) {
	t.Helper()
	if result.OK() {
		t.Fatalf("expected validation failure containing %q", text)
	}
	if !strings.Contains(result.Error(), text) {
		t.Fatalf("expected diagnostic containing %q, got %q", text, result.Error())
	}
}
