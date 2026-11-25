package main

import (
	"fmt"
	"io"
	"strings"
)

func (cmd command) completion(shell string) error {
	if shell != "bash" {
		return fmt.Errorf("unsupported completion shell: %s (supported: bash)", shell)
	}
	var script strings.Builder
	fmt.Fprint(&script, `_circuit_complete() {
    local i word path options choices=''
    local -a words=()
    for ((i=1; i<COMP_CWORD; i++)); do
        word=${COMP_WORDS[i]}
        case "$word" in
            --) return ;;
            --task)
                ((++i))
                if ((i == COMP_CWORD)); then return; fi
                ;;
            -*) ;;
            *) words+=("$word") ;;
        esac
    done
    path=${words[*]}
    options='-h --help -q --quiet -v --verbose --json'
    case "$path" in
`)
	for _, noun := range []actionName{actionRoot, actionMachine, actionSession} {
		fmt.Fprintf(&script, "        '%s') choices='%s' ;;\n", noun, strings.Join(subcommands(noun), " "))
	}
	fmt.Fprint(&script, `        'completion') choices='bash'; options="${options% --json}" ;;
        'completion bash') options="${options% --json}" ;;
        'session drive'|'session drive '*) options+=' --task' ;;
    esac
    if [[ ${COMP_WORDS[COMP_CWORD]} == -* ]]; then choices=$options; fi
    COMPREPLY=()
    while IFS= read -r word; do
        COMPREPLY+=("$word")
    done < <(compgen -W "$choices" -- "${COMP_WORDS[COMP_CWORD]}")
}
complete -F _circuit_complete circuit
`)
	_, err := io.WriteString(cmd.stdout, script.String())
	return err
}

func subcommands(noun actionName) []string {
	var names []string
	for _, spec := range commandTree {
		parent, verb, hasParent := strings.Cut(string(spec.action), " ")
		switch {
		case noun == actionRoot && !hasParent:
			names = append(names, parent)
		case noun != actionRoot && hasParent && string(noun) == parent:
			names = append(names, verb)
		}
	}
	return names
}
