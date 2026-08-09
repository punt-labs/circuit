package circuitb

import "os"

func LoadFile(path string) (Machine, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Machine{}, err
	}
	tokens, err := lex(path, string(content))
	if err != nil {
		return Machine{}, err
	}
	raw, err := parse(tokens)
	if err != nil {
		return Machine{}, err
	}
	if err := validateProfile(raw); err != nil {
		return Machine{}, err
	}
	return resolve(raw)
}
