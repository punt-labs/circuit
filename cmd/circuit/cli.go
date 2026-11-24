package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	actionRoot         actionName = ""
	actionHelp         actionName = "help"
	actionUnknown      actionName = "unknown"
	actionMachine      actionName = "machine"
	actionMachineList  actionName = "machine list"
	actionMachineCheck actionName = "machine check"
	actionScaffold     actionName = "machine scaffold"
	actionSession      actionName = "session"
	actionStart        actionName = "session start"
	actionSessionList  actionName = "session list"
	actionSessionShow  actionName = "session show"
	actionAdvance      actionName = "session advance"
	actionStop         actionName = "session stop"
	actionRemove       actionName = "session remove"
	actionDrive        actionName = "session drive"
	actionVersion      actionName = "version"
	actionStatus       actionName = "status"
	actionDoctor       actionName = "doctor"
	actionInstall      actionName = "install"
	actionCompletion   actionName = "completion"
)

type commandSpec struct {
	action   actionName
	usage    string
	purpose  string
	success  string
	min, max int
}

const machineOperand = "<machine>"

var commandTree = []commandSpec{
	{actionMachine, "<command>", "Manage B machines.", "", 0, 0},
	{actionMachineList, "", "List available B machines.", "Enumerated machine names in machines/ (JSON array of strings).", 0, 0},
	{actionMachineCheck, machineOperand, "Validate a named B machine and its check bindings.", "Validated the machine without starting a session (JSON machine and checks).", 1, 1},
	{actionScaffold, machineOperand, "Generate missing check bindings and false stubs for a named B machine.", "Generated any missing bindings and stubs (JSON machine and generated IDs).", 1, 1},
	{actionSession, "<command>", "Manage workflow sessions.", "", 0, 0},
	{actionStart, machineOperand, "Start a session for a named B machine.", "Started and persisted the session (JSON session detail).", 1, 1},
	{actionSessionList, "", "List active and stopped sessions in this workspace.", "Read all session reports (JSON array of session details).", 0, 0},
	{actionSessionShow, "<session>", "Show a session by ID.", "Read the specified session report (JSON session detail).", 1, 1},
	{actionAdvance, "<event> [session]", "Request an event by name; defaults to the only active session.", "Evaluated the event and persisted its outcome (JSON allowed, states, and failed predicates if blocked). A blocked event does not advance the session and is not queued.", 1, 2},
	{actionStop, "[session]", "Stop a session by ID; defaults to the only active session, or a lone stopped session.", "The selected session is stopped (JSON session detail).", 0, 1},
	{actionRemove, "<session>", "Remove a stopped session by ID.", "Removed the stopped session (JSON session ID and final lifecycle state).", 1, 1},
	{actionDrive, "<machine> [--task <goal>]", "Run a named B machine to completion with Pi.", "Reached a terminal state and persisted the completed session (JSON session detail).", 1, 1},
	{actionVersion, "", "Print the Circuit version.", "Read the binary version (JSON version).", 0, 0},
	{actionStatus, "", "Summarize operating state for this workspace.", "Read the number of active and stopped sessions (JSON active and stopped counts).", 0, 0},
	{actionDoctor, "", "Check Circuit installation health.", "All required checks passed (JSON checks and overall ok). Optional Pi and Git checks do not change the exit code.", 0, 0},
	{actionInstall, "", "Install this executable to ~/.local/bin/circuit.", "Installed the running binary atomically (JSON installed path).", 0, 0},
	{actionCompletion, "<shell>", "Generate a shell completion script (shell: bash).", "Wrote a completion script to stdout; --json is not supported.", 1, 1},
}

type invocation struct {
	action  actionName
	args    []string
	task    string
	taskSet bool
	help    bool
	json    bool
	quiet   bool
	verbose bool
}

func parseInvocation(args []string) (invocation, error) {
	call := invocation{action: actionRoot}
	var operands []string
	var parseErr error
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--":
			operands = append(operands, args[i+1:]...)
			i = len(args)
		case "--help", "-h":
			call.help = true
		case jsonFlag:
			call.json = true
		case "--verbose", "-v":
			call.verbose = true
		case "--quiet", "-q":
			call.quiet = true
		case "--task":
			consumed, err := call.parseTask(operands, args[i+1:])
			i += consumed
			if err != nil {
				parseErr = err
			}
		default:
			if strings.HasPrefix(arg, "-") {
				parseErr = fmt.Errorf("unknown option: %s", arg)
			} else {
				operands = append(operands, arg)
			}
		}
	}
	spec, rest, err := resolveCommand(operands)
	call.action, call.args = spec.action, rest
	if parseErr != nil {
		return call, parseErr
	}
	if err != nil {
		return call, err
	}
	return call.validate(spec)
}

func (call *invocation) parseTask(path, args []string) (int, error) {
	if len(path) < 2 || strings.Join(path[:2], " ") != string(actionDrive) {
		return 0, errors.New("--task is only supported after session drive")
	}
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return 0, errors.New("--task requires a goal")
	}
	if call.taskSet {
		return 1, errors.New("--task may only be specified once")
	}
	call.task, call.taskSet = args[0], true
	return 1, nil
}

func (call invocation) validate(spec commandSpec) (invocation, error) {
	if call.verbose && call.quiet {
		return call, errors.New("--verbose and --quiet are mutually exclusive")
	}
	if call.json && call.action == actionCompletion {
		return call, errors.New("completion writes a shell script and does not support --json")
	}
	if call.help || spec.success == "" {
		call.help = true
		return call, nil
	}
	if len(call.args) < spec.min || len(call.args) > spec.max {
		return call, fmt.Errorf("usage: circuit %s", spec.syntax())
	}
	return call, nil
}

func resolveCommand(operands []string) (commandSpec, []string, error) {
	if len(operands) == 0 {
		return commandSpec{action: actionRoot}, nil, nil
	}
	path := operands[0]
	depth := 1
	if (path == string(actionMachine) || path == string(actionSession)) && len(operands) > 1 && !strings.Contains(operands[1], " ") {
		path += " " + operands[1]
		depth++
	}
	for _, spec := range commandTree {
		if string(spec.action) == path && (depth == 2 || !strings.Contains(path, " ")) && (spec.success != "" || len(operands) == depth) {
			return spec, operands[depth:], nil
		}
	}
	return commandSpec{action: actionUnknown}, nil, fmt.Errorf("unknown command: %s", path)
}

func (spec commandSpec) syntax() string {
	return strings.TrimSpace(string(spec.action) + " " + spec.usage)
}

func (cmd command) run(args []string) error {
	call, err := parseInvocation(args)
	return cmd.runInvocation(call, err)
}

func (cmd command) runInvocation(call invocation, err error) error {
	started := time.Now()
	cmd.json, cmd.quiet = call.json, call.quiet
	var blocked bool
	action := call.action
	if err == nil {
		if call.help {
			action = actionHelp
			err = cmd.printHelp(call.action)
		} else {
			blocked, err = cmd.dispatchWithSessionLock(call)
		}
	} else if action == actionRoot {
		action = actionUnknown
	}
	if err != nil && call.json {
		var result any = map[string]string{"error": err.Error()}
		var doctorErr *doctorReport
		if errors.As(err, &doctorErr) {
			result = doctorErr
		}
		if outputErr := cmd.writeJSON(result); outputErr != nil {
			err = errors.Join(err, outputErr)
		}
	}
	if cmd.observe != nil {
		cmd.observe.record(action, blocked, err, time.Since(started))
	}
	if call.verbose && !call.quiet {
		fmt.Fprintf(cmd.stderr, "circuit: %s: %s (%s)\n", action, outcomeFor(blocked, err), time.Since(started).Round(time.Millisecond)) //nolint:errcheck // optional diagnostics do not change the result
	}
	return err
}

func (cmd command) printHelp(action actionName) error {
	var help strings.Builder
	printHelpCommands(&help, action)
	if action == actionDrive {
		fmt.Fprintln(&help, "\nOptions:\n  --task <goal>  Goal text passed to Pi (default: no explicit goal).")
	}
	fmt.Fprintln(&help, "\nGlobal flags (off by default):\n  -h, --help     Show help for this command.\n  -q, --quiet    Suppress progress and nonessential diagnostics.\n  -v, --verbose  Log command outcome and duration to stderr (incompatible with quiet).")
	if action != actionCompletion {
		fmt.Fprintln(&help, "      --json     Write one JSON result, or an object with an error string on failure.")
	}
	if action == actionRoot || action == actionMachine || action == actionSession {
		fmt.Fprintln(&help, "\nRun a command with --help for its exit status and output contract.")
	}
	if cmd.json {
		return cmd.writeJSON(map[string]string{"help": help.String()})
	}
	_, err := io.WriteString(cmd.stdout, help.String())
	return err
}

func printHelpCommands(help *strings.Builder, action actionName) {
	switch action {
	case actionRoot:
		fmt.Fprintln(help, "Circuit runs workflows defined by B machines.\n\nusage: circuit [flags] <command>\n\nCommands:")
	case actionMachine, actionSession:
		for _, spec := range commandTree {
			if spec.action == action {
				fmt.Fprintf(help, "%s\n\nusage: circuit %s\n\nCommands:\n", spec.purpose, spec.syntax())
			}
		}
	default:
	}
	for _, spec := range commandTree {
		name := string(spec.action)
		switch {
		case action == actionRoot && !strings.Contains(name, " "):
			fmt.Fprintf(help, "  %-12s %s\n", name, spec.purpose)
		case action != actionRoot && strings.HasPrefix(name, string(action)+" "):
			fmt.Fprintf(help, "  %-12s %s\n", strings.TrimPrefix(name, string(action)+" "), spec.purpose)
		case spec.action == action && spec.success != "":
			fmt.Fprintf(help, "%s\n\nusage: circuit %s\n\nExit 0: %s\n", spec.purpose, spec.syntax(), spec.success)
		}
	}
}
