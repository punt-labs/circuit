package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/punt-labs/circuit/internal/circuitb"
	"github.com/punt-labs/circuit/internal/circuitrpc"
	"github.com/punt-labs/circuit/internal/circuitrun"
	"gopkg.in/yaml.v3"
)

const (
	exitUsage     = 2
	jsonFlag      = "--json"
	traceKeyState = "state"
	traceKeyType  = "type"
)

type command struct {
	stdout  io.Writer
	stderr  io.Writer
	cwd     string
	backend circuitrpc.AgentBackend
}

func main() {
	cmd := command{stdout: os.Stdout, stderr: os.Stderr}
	if err := cmd.run(os.Args[1:]); err != nil {
		fmt.Fprintln(cmd.stderr, err)
		os.Exit(exitUsage)
	}
}

func (cmd command) run(args []string) error {
	if len(args) == 0 {
		cmd.printUsage()
		return errors.New("missing command")
	}

	switch args[0] {
	case "help", "--help", "-h":
		cmd.printUsage()
		return nil
	case "list":
		return cmd.listMachines(args[1:])
	case "load":
		return cmd.load(args[1:])
	case "scaffold":
		return cmd.scaffold(args[1:])
	case "start":
		return cmd.start(args[1:])
	case "status":
		return cmd.status(args[1:])
	case "advance":
		return cmd.advance(args[1:])
	case "stop":
		return cmd.stop(args[1:])
	case "unload":
		return cmd.unload(args[1:])
	case "drive":
		return cmd.drive(args[1:])
	default:
		cmd.printUsage()
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func (cmd command) listMachines(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("expected no arguments, got %d", len(args))
	}
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	names, err := runtime.ListMachines()
	if err != nil {
		return err
	}
	for _, name := range names {
		fmt.Fprintln(cmd.stdout, name)
	}
	return nil
}

func (cmd command) load(args []string) error {
	jsonMode, machine, err := parseLoadArgs(args)
	if err != nil {
		return err
	}
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	report, err := runtime.Load(machine)
	if err != nil {
		return err
	}
	if jsonMode {
		return cmd.writeLoadJSON(report)
	}
	fmt.Fprintf(cmd.stdout, "loaded: %s\n", report.MachineName)
	cmd.printCheckBindings(report.Checks)
	return nil
}

func parseJSONFlagWithMachineArg(args []string) (bool, string, error) {
	var jsonMode bool
	var machine string
	for _, arg := range args {
		switch arg {
		case jsonFlag:
			jsonMode = true
		default:
			if machine != "" {
				return false, "", fmt.Errorf("unexpected argument: %s", arg)
			}
			machine = arg
		}
	}
	if machine == "" {
		return false, "", fmt.Errorf("expected one machine argument, got %d arguments", len(args))
	}
	return jsonMode, machine, nil
}

func parseLoadArgs(args []string) (bool, string, error) {
	return parseJSONFlagWithMachineArg(args)
}

func (cmd command) scaffold(args []string) error {
	jsonMode, machine, err := parseScaffoldArgs(args)
	if err != nil {
		return err
	}
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	report, err := runtime.Scaffold(machine)
	if err != nil {
		return err
	}
	if jsonMode {
		return cmd.writeScaffoldJSON(report)
	}
	fmt.Fprintf(cmd.stdout, "scaffolded: %s\n", report.MachineName)
	for _, variable := range report.GeneratedBindings {
		fmt.Fprintf(cmd.stdout, "binding: %s\n", variable)
	}
	for _, check := range report.GeneratedRegistryIDs {
		fmt.Fprintf(cmd.stdout, "stub: %s -> false\n", check)
	}
	return nil
}

func parseScaffoldArgs(args []string) (bool, string, error) {
	return parseJSONFlagWithMachineArg(args)
}

func (cmd command) start(args []string) error {
	jsonMode, machine, err := parseStartArgs(args)
	if err != nil {
		return err
	}
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	id, report, err := runtime.Start(machine)
	if err != nil {
		return err
	}
	if err := runtime.Suspend(); err != nil {
		return err
	}
	if jsonMode {
		return cmd.writeStatusReportJSON(report)
	}
	fmt.Fprintf(cmd.stdout, "started: %s\n", report.MachineName)
	fmt.Fprintf(cmd.stdout, "session: %s\n", id)
	cmd.printStatusReport(report)
	return nil
}

func parseStartArgs(args []string) (bool, string, error) {
	return parseJSONFlagWithMachineArg(args)
}

func (cmd command) drive(args []string) error {
	machine, task, err := parseDriveArgs(args)
	if err != nil {
		return err
	}
	backend, stopBackend, err := cmd.agentBackend()
	if err != nil {
		return err
	}
	if stopBackend != nil {
		defer stopBackend()
	}
	runtime, id, trace, err := cmd.startDrivenSession(machine)
	if err != nil {
		return err
	}
	defer func() {
		if err := trace.Close(); err != nil {
			fmt.Fprintf(cmd.stderr, "close trace: %v\n", err)
		}
	}()
	backend = traceBackend{backend: backend, trace: trace, cwd: cmd.workingDir()}
	guidance, err := cmd.loadGuidance(machine, task)
	if err != nil {
		return err
	}
	result, err := circuitrpc.RunGuidedSession(runtime, backend, guidance)
	cmd.printDriveTransitions(trace, result.Transitions)
	if err != nil {
		return err
	}
	return cmd.finishDrivenSession(runtime, id)
}

func (cmd command) startDrivenSession(machine string) (*circuitrun.Runtime, string, driveTrace, error) {
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return nil, "", nil, err
	}
	id, report, err := runtime.Start(machine)
	if err != nil {
		return nil, "", nil, err
	}
	fmt.Fprintf(cmd.stdout, "started: %s\n", report.MachineName)
	fmt.Fprintf(cmd.stdout, "session: %s\n", id)
	trace, err := cmd.openDriveTrace(id)
	return runtime, id, trace, err
}

func (cmd command) printDriveTransitions(trace io.Writer, transitions []circuitrun.AdvanceReport) {
	for _, transition := range transitions {
		writeDriveTransition(trace, transition)
		if transition.Allowed {
			fmt.Fprintf(cmd.stdout, "advanced: %s -> %s\n", transition.From, transition.To)
		}
	}
}

func writeDriveTransition(trace io.Writer, transition circuitrun.AdvanceReport) {
	writeTrace(trace, map[string]any{traceKeyType: "advance", traceKeyState: transition.From, "event": transition.Event, "allowed": transition.Allowed, "from": transition.From, "to": transition.To, "failed": transition.Failed})
}

func (cmd command) finishDrivenSession(runtime *circuitrun.Runtime, id string) error {
	status, err := runtime.StatusByID(id)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.stdout, "terminal: %s\n", status.Current)
	return runtime.Suspend()
}

type driveTrace interface {
	io.Writer
	Close() error
}

type traceBackend struct {
	backend circuitrpc.AgentBackend
	trace   io.Writer
	cwd     string
}

func (backend traceBackend) Prompt(message string) (string, error) {
	state := currentStateFromPrompt(message)
	writeTrace(backend.trace, map[string]any{traceKeyType: "prompt", traceKeyState: state, "text": message})
	response, err := backend.backend.Prompt(message)
	writeTrace(backend.trace, map[string]any{traceKeyType: "response", traceKeyState: state, "text": response})
	writeTrace(backend.trace, map[string]any{traceKeyType: "workspace", traceKeyState: state, "status": gitStatusShort(backend.cwd)})
	return response, err
}

func (cmd command) openDriveTrace(sessionID string) (driveTrace, error) {
	path := filepath.Join(cmd.workingDir(), ".tmp", "circuit", sessionID, "drive.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil { //nolint:gosec // G703: path is session-scoped under .tmp/circuit/
		return nil, err
	}
	return os.Create(path) //nolint:gosec // G304: path is session-scoped trace file under .tmp/circuit/
}

func writeTrace(writer io.Writer, event map[string]any) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	fmt.Fprintf(writer, "%s\n", data)
}

func currentStateFromPrompt(message string) string {
	for line := range strings.SplitSeq(message, "\n") {
		if after, ok := strings.CutPrefix(line, "Current state: "); ok {
			return after
		}
	}
	return ""
}

func gitStatusShort(cwd string) string {
	output, err := exec.Command("git", "-C", cwd, "status", "--short").Output() //nolint:gosec // G204: cwd is the project working directory
	if err != nil {
		return ""
	}
	return string(output)
}

func (cmd command) agentBackend() (circuitrpc.AgentBackend, func(), error) {
	if cmd.backend != nil {
		return cmd.backend, nil, nil
	}
	return launchPiBackend(cmd.workingDir())
}

func parseDriveArgs(args []string) (string, string, error) {
	var machine, task string
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--task":
			if index+1 >= len(args) {
				return "", "", errors.New("--task requires a value")
			}
			task = args[index+1]
			index++
		default:
			if machine != "" {
				return "", "", fmt.Errorf("unexpected argument: %s", arg)
			}
			machine = arg
		}
	}
	if machine == "" {
		return "", "", errors.New("drive requires a machine name")
	}
	return machine, task, nil
}

func launchPiBackend(cwd string) (circuitrpc.AgentBackend, func(), error) {
	process := exec.Command("pi", "--mode", "rpc", "--no-session", "--approve")
	process.Dir = cwd
	process.Stderr = os.Stderr
	stdin, err := process.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	stdout, err := process.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := process.Start(); err != nil {
		return nil, nil, err
	}
	backend := circuitrpc.NewPiRPCBackend(stdin, bufio.NewReader(stdout))
	cleanup := func() {
		if err := stdin.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "close stdin: %v\n", err)
		}
		if err := process.Process.Signal(os.Interrupt); err != nil {
			fmt.Fprintf(os.Stderr, "signal process: %v\n", err)
		}
		if err := process.Wait(); err != nil {
			fmt.Fprintf(os.Stderr, "wait process: %v\n", err)
		}
	}
	return backend, cleanup, nil
}

type guidanceFile struct {
	States map[string]circuitrpc.StateGuidance `yaml:"states"`
}

func (cmd command) loadGuidance(machine string, task string) (circuitrpc.DriverGuidance, error) {
	path := filepath.Join(cmd.workingDir(), "machines", machine+".prompts.yaml")
	content, err := os.ReadFile(path) //nolint:gosec // G304: path is machine-local prompts file
	if err != nil {
		return circuitrpc.DriverGuidance{}, err
	}
	var file guidanceFile
	if err := yaml.Unmarshal(content, &file); err != nil {
		return circuitrpc.DriverGuidance{}, err
	}
	return circuitrpc.DriverGuidance{Goal: task, States: file.States}, nil
}

func (cmd command) status(args []string) error {
	jsonMode, session, err := parseStatusArgs(args)
	if err != nil {
		return err
	}
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	if session != "" {
		report, err := runtime.StatusByID(session)
		if err != nil {
			return err
		}
		if jsonMode {
			return cmd.writeStatusJSON([]circuitrun.StatusReport{report})
		}
		cmd.printStatusReport(report)
		return runtime.Suspend()
	}
	reports, err := runtime.StatusAll()
	if err != nil {
		return err
	}
	if jsonMode {
		return cmd.writeStatusJSON(reports)
	}
	if len(reports) == 0 {
		fmt.Fprintln(cmd.stdout, "no session")
		return nil
	}
	cmd.printStatusReports(reports)
	return runtime.Suspend()
}

func parseStatusArgs(args []string) (bool, string, error) {
	return parseJSONFlagWithOptionalArgument(args)
}

type loadJSONEntry struct {
	Machine string                          `json:"machine"`
	Checks  []circuitrun.CheckBindingReport `json:"checks"`
}

type scaffoldJSONEntry struct {
	Machine              string   `json:"machine"`
	GeneratedBindings    []string `json:"generatedBindings"`
	GeneratedRegistryIDs []string `json:"generatedRegistryIDs"`
}

type advanceJSONEntry struct {
	Session string               `json:"session,omitempty"`
	Allowed bool                 `json:"allowed"`
	From    string               `json:"from"`
	To      string               `json:"to,omitempty"`
	Event   string               `json:"event"`
	Failed  []string             `json:"failed,omitempty"`
	Checks  map[string]checkJSON `json:"checks,omitempty"`
}

type statusJSONEntry struct {
	Session      string               `json:"session"`
	SessionState string               `json:"sessionState"`
	Machine      string               `json:"machine"`
	Current      string               `json:"current"`
	Enabled      []callStatusJSON     `json:"enabled"`
	Blocked      []callStatusJSON     `json:"blocked"`
	Checks       map[string]checkJSON `json:"checks,omitempty"`
}

type unloadJSONEntry struct {
	Session      string `json:"session"`
	SessionState string `json:"sessionState"`
}

type callStatusJSON struct {
	Call   string   `json:"call"`
	Failed []string `json:"failed,omitempty"`
}

type checkJSON struct {
	Result      bool `json:"result"`
	Invocations int  `json:"invocations"`
}

func (cmd command) writeLoadJSON(report circuitrun.LoadReport) error {
	return cmd.writeJSON(loadJSONEntry{Machine: report.MachineName, Checks: report.Checks})
}

func (cmd command) writeScaffoldJSON(report circuitrun.ScaffoldReport) error {
	entry := scaffoldJSONEntry{
		Machine:              report.MachineName,
		GeneratedBindings:    report.GeneratedBindings,
		GeneratedRegistryIDs: report.GeneratedRegistryIDs,
	}
	return cmd.writeJSON(entry)
}

func (cmd command) writeAdvanceJSON(report circuitrun.AdvanceReport) error {
	checks := make(map[string]checkJSON, len(report.Checks))
	for name, check := range report.Checks {
		checks[name] = checkJSON{Result: check.LastResult, Invocations: check.Invocations}
	}
	entry := advanceJSONEntry{
		Session: report.SessionID,
		Allowed: report.Allowed,
		From:    report.From,
		To:      report.To,
		Event:   report.Event,
		Failed:  report.Failed,
	}
	if len(checks) > 0 {
		entry.Checks = checks
	}
	return cmd.writeJSON(entry)
}

func (cmd command) writeStatusReportJSON(report circuitrun.StatusReport) error {
	return cmd.writeJSON(statusJSONEntryFrom(report))
}

func (cmd command) writeStatusJSON(reports []circuitrun.StatusReport) error {
	entries := make([]statusJSONEntry, 0, len(reports))
	for _, report := range reports {
		entries = append(entries, statusJSONEntryFrom(report))
	}
	return cmd.writeJSON(entries)
}

func (cmd command) writeJSON(entry any) error {
	output, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.stdout, string(output))
	return nil
}

func statusJSONEntryFrom(report circuitrun.StatusReport) statusJSONEntry {
	enabled := callStatusJSONSliceFrom(report.Enabled)
	blocked := callStatusJSONSliceFrom(report.Blocked)
	checks := make(map[string]checkJSON, len(report.Checks))
	for name, check := range report.Checks {
		checks[name] = checkJSON{Result: check.LastResult, Invocations: check.Invocations}
	}
	entry := statusJSONEntry{
		Session:      report.SessionID,
		SessionState: string(report.SessionState),
		Machine:      report.MachineName,
		Current:      report.Current,
		Enabled:      enabled,
		Blocked:      blocked,
	}
	if len(checks) > 0 {
		entry.Checks = checks
	}
	return entry
}

func callStatusJSONSliceFrom(calls []circuitb.CallStatus) []callStatusJSON {
	result := make([]callStatusJSON, 0, len(calls))
	for _, call := range calls {
		result = append(result, callStatusJSON{Call: call.Call, Failed: call.Failed})
	}
	return result
}

func (cmd command) advance(args []string) error {
	jsonMode, event, session, err := parseAdvanceArgs(args)
	if err != nil {
		return err
	}
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	var report circuitrun.AdvanceReport
	if session != "" {
		report, err = runtime.AdvanceByID(session, event)
	} else {
		report, err = runtime.Advance(event)
	}
	if err != nil {
		return err
	}
	if err := runtime.Suspend(); err != nil {
		return err
	}
	if jsonMode {
		return cmd.writeAdvanceJSON(report)
	}
	if !report.Allowed {
		fmt.Fprintf(cmd.stdout, "blocked: Advance(%s)\n", report.Event)
		for _, failed := range report.Failed {
			fmt.Fprintf(cmd.stdout, "  needs: %s\n", failed)
		}
		cmd.printChecks(report.Checks)
		return nil
	}
	fmt.Fprintf(cmd.stdout, "advanced: %s -> %s\n", report.From, report.To)
	cmd.printChecks(report.Checks)
	return nil
}

func parseAdvanceArgs(args []string) (bool, string, string, error) {
	var jsonMode bool
	var event, session string
	for _, arg := range args {
		switch arg {
		case jsonFlag:
			jsonMode = true
		default:
			switch {
			case event == "":
				event = arg
			case session == "":
				session = arg
			default:
				return false, "", "", fmt.Errorf("unexpected argument: %s", arg)
			}
		}
	}
	if event == "" {
		return false, "", "", fmt.Errorf("expected one event and optional session, got %d arguments", len(args))
	}
	return jsonMode, event, session, nil
}

func (cmd command) unload(args []string) error {
	jsonMode, session, err := parseUnloadArgs(args)
	if err != nil {
		return err
	}
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	if err := runtime.UnloadByID(session); err != nil {
		return err
	}
	if jsonMode {
		return cmd.writeJSON(unloadJSONEntry{Session: session, SessionState: string(circuitrun.SessionUnloaded)})
	}
	fmt.Fprintln(cmd.stdout, "unloaded")
	return nil
}

func parseUnloadArgs(args []string) (bool, string, error) {
	jsonMode, session, err := parseJSONFlagWithOptionalArgument(args)
	if err != nil {
		return false, "", err
	}
	if session == "" {
		return false, "", fmt.Errorf("expected one session argument, got %d arguments", len(args))
	}
	return jsonMode, session, nil
}

func (cmd command) stop(args []string) error {
	jsonMode, session, err := parseStopArgs(args)
	if err != nil {
		return err
	}
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	if session != "" {
		err = runtime.StopByID(session)
	} else {
		err = runtime.Stop()
	}
	if err != nil {
		return err
	}
	if jsonMode {
		var report circuitrun.StatusReport
		if session != "" {
			report, err = runtime.StatusByID(session)
		} else {
			report, err = runtime.Status()
		}
		if err != nil {
			return err
		}
		return cmd.writeStatusReportJSON(report)
	}
	fmt.Fprintln(cmd.stdout, "stopped")
	return nil
}

func parseStopArgs(args []string) (bool, string, error) {
	return parseJSONFlagWithOptionalArgument(args)
}

func parseJSONFlagWithOptionalArgument(args []string) (bool, string, error) {
	var jsonMode bool
	var argument string
	for _, arg := range args {
		switch arg {
		case jsonFlag:
			jsonMode = true
		default:
			if argument != "" {
				return false, "", fmt.Errorf("unexpected argument: %s", arg)
			}
			argument = arg
		}
	}
	return jsonMode, argument, nil
}

func (cmd command) printStatusReports(reports []circuitrun.StatusReport) {
	for index, report := range reports {
		if index > 0 {
			fmt.Fprintln(cmd.stdout)
		}
		cmd.printStatusReport(report)
	}
}

func (cmd command) printStatusReport(report circuitrun.StatusReport) {
	fmt.Fprintf(cmd.stdout, "session: %s\n", report.SessionID)
	fmt.Fprintf(cmd.stdout, "session-state: %s\n", report.SessionState)
	fmt.Fprintf(cmd.stdout, "machine: %s\n", report.MachineName)
	fmt.Fprintf(cmd.stdout, "current: %s\n", report.Current)
	fmt.Fprintln(cmd.stdout, "enabled:")
	for _, call := range report.Enabled {
		fmt.Fprintf(cmd.stdout, "  %s\n", call.Call)
	}
	fmt.Fprintln(cmd.stdout, "blocked:")
	for _, call := range report.Blocked {
		fmt.Fprintf(cmd.stdout, "  %s\n", call.Call)
	}
	cmd.printChecks(report.Checks)
}

func (cmd command) printCheckBindings(checks []circuitrun.CheckBindingReport) {
	if len(checks) == 0 {
		return
	}
	fmt.Fprintln(cmd.stdout, "checks:")
	for _, check := range checks {
		fmt.Fprintf(cmd.stdout, "  %s -> %s: %s\n", check.Variable, check.Use, check.Returns)
	}
}

func (cmd command) printChecks(checks map[string]circuitrun.CheckRuntime) {
	if len(checks) == 0 {
		return
	}
	names := make([]string, 0, len(checks))
	for name := range checks {
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Fprintln(cmd.stdout, "checks:")
	for _, name := range names {
		check := checks[name]
		fmt.Fprintf(cmd.stdout, "  %s: %s (invocations: %d)\n", name, formatBool(check.LastResult), check.Invocations)
	}
}

func formatBool(value bool) string {
	if value {
		return "TRUE"
	}
	return "FALSE"
}

func (cmd command) printUsage() {
	fmt.Fprintln(cmd.stderr, "usage: circuit <command> [args]")
	fmt.Fprintln(cmd.stderr, "")
	fmt.Fprintln(cmd.stderr, "commands:")
	fmt.Fprintln(cmd.stderr, "  list                        list available B machines")
	fmt.Fprintln(cmd.stderr, "  load [--json] <machine>     validate machine and check bindings")
	fmt.Fprintln(cmd.stderr, "  scaffold [--json] <machine> generate missing check bindings and false stubs")
	fmt.Fprintln(cmd.stderr, "  start <machine>             start a circuit session")
	fmt.Fprintln(cmd.stderr, "  status [--json] [session]   print circuit session status")
	fmt.Fprintln(cmd.stderr, "  advance [--json] <event> [session]")
	fmt.Fprintln(cmd.stderr, "                              apply Advance(event) to a circuit session")
	fmt.Fprintln(cmd.stderr, "  stop [session]              stop a circuit session")
	fmt.Fprintln(cmd.stderr, "  unload [--json] <session>   remove a stopped circuit session")
	fmt.Fprintln(cmd.stderr, "  drive <machine> [--task s]  run a machine end-to-end against an agent backend")
}

func (cmd command) workingDir() string {
	if cmd.cwd != "" {
		return cmd.cwd
	}
	return "."
}
