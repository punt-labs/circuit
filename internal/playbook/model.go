package playbook

// Playbook is the top-level playbook document.
type Playbook struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Mode        string      `yaml:"mode"`
	Steps       []Action    `yaml:"steps"`
	States      []State     `yaml:"states"`
	Instance    string      `yaml:"instance"`
	Parameters  []Parameter `yaml:"parameters"`
}

type Parameter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Default     string `yaml:"default"`
}

type State struct {
	ID          string       `yaml:"id"`
	Description string       `yaml:"description"`
	Poll        string       `yaml:"poll"`
	Actions     []Action     `yaml:"actions"`
	Transitions []Transition `yaml:"transitions"`
}

type Action struct {
	ID            string `yaml:"id"`
	Type          string `yaml:"type"`
	Description   string `yaml:"description"`
	Script        string `yaml:"script"`
	Check         string `yaml:"check"`
	Context       string `yaml:"context"`
	Postcondition string `yaml:"postcondition"`
	OnFailure     string `yaml:"on_failure"`
}

type Transition struct {
	To   string       `yaml:"to"`
	When []GuardCheck `yaml:"when"`
}

type GuardCheck struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Check       string `yaml:"check"`
	Context     string `yaml:"context"`
}

func (playbook Playbook) IsMachine() bool {
	return len(playbook.States) > 0
}

func (playbook Playbook) IsLinear() bool {
	return len(playbook.Steps) > 0
}
