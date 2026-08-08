package playbook

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func ParseFile(path string) (Playbook, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Playbook{}, fmt.Errorf("read playbook: %w", err)
	}
	return Parse(content)
}

func Parse(content []byte) (Playbook, error) {
	var playbook Playbook
	if err := yaml.Unmarshal(content, &playbook); err != nil {
		return Playbook{}, fmt.Errorf("parse playbook: %w", err)
	}
	return playbook, nil
}
