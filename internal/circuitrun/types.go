package circuitrun

import "github.com/punt-labs/circuit/internal/circuitb"

type SessionState string

const (
	SessionUnloaded  SessionState = "unloaded"
	SessionActive    SessionState = "active"
	SessionSuspended SessionState = "suspended"
	SessionStopped   SessionState = "stopped"
)

type Runtime struct {
	root      string
	sessions  map[string]*Run
	currentID string
	lastState SessionState
}

type Run struct {
	SessionID   string                  `json:"sessionId"`
	MachineName string                  `json:"machineName"`
	MachineFile string                  `json:"machineFile"`
	Current     string                  `json:"current"`
	Session     SessionState            `json:"session"`
	Booleans    map[string]bool         `json:"booleans,omitempty"`
	Checks      map[string]CheckRuntime `json:"checks,omitempty"`
}

type StatusReport struct {
	SessionID    string
	SessionState SessionState
	MachineName  string
	Current      string
	Enabled      []circuitb.CallStatus
	Blocked      []circuitb.CallStatus
	Checks       map[string]CheckRuntime
}

type AdvanceReport struct {
	SessionID string
	Allowed   bool
	From      string
	To        string
	Event     string
	Failed    []string
	Checks    map[string]CheckRuntime
}

type CheckRuntime struct {
	Invocations int  `json:"invocations"`
	LastResult  bool `json:"lastResult"`
}

type LoadReport struct {
	MachineName string
	Checks      []CheckBindingReport
}

type CheckBindingReport struct {
	Variable string
	Use      string
	Returns  string
}

type ScaffoldReport struct {
	MachineName          string
	GeneratedBindings    []string
	GeneratedRegistryIDs []string
}

type checkBindingsFile struct {
	Checks map[string]checkBinding `yaml:"checks"`
}

type checkBinding struct {
	Use string `yaml:"use"`
}

type checkRegistryFile struct {
	Checks map[string]registeredCheck `yaml:"checks"`
}

type registeredCheck struct {
	Kind    string `yaml:"kind"`
	Command string `yaml:"command"`
	Returns string `yaml:"returns"`
}
