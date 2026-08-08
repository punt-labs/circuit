package playbook

import "fmt"

type Summary struct {
	Name        string
	Kind        string
	States      int
	Steps       int
	Transitions int
	Initial     string
	Terminals   int
}

func Summarize(playbook Playbook) Summary {
	summary := Summary{Name: playbook.Name, Steps: len(playbook.Steps), States: len(playbook.States)}
	if playbook.IsMachine() {
		summary.Kind = "machine"
		if len(playbook.States) > 0 {
			summary.Initial = playbook.States[0].ID
		}
		for _, state := range playbook.States {
			summary.Transitions += len(state.Transitions)
			if len(state.Transitions) == 0 {
				summary.Terminals++
			}
		}
		return summary
	}
	summary.Kind = "linear"
	return summary
}

func (summary Summary) Lines() []string {
	lines := []string{
		fmt.Sprintf("name: %s", summary.Name),
		fmt.Sprintf("kind: %s", summary.Kind),
	}
	if summary.Kind == "machine" {
		lines = append(lines,
			fmt.Sprintf("states: %d", summary.States),
			fmt.Sprintf("initial: %s", summary.Initial),
			fmt.Sprintf("transitions: %d", summary.Transitions),
			fmt.Sprintf("terminal_states: %d", summary.Terminals),
		)
		return lines
	}
	lines = append(lines, fmt.Sprintf("steps: %d", summary.Steps))
	return lines
}
