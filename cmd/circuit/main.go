package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/punt-labs/circuit/internal/circuitb"
	"github.com/punt-labs/circuit/internal/circuitrpc"
	"github.com/punt-labs/circuit/internal/circuitrun"
	"github.com/punt-labs/punt-observe-go/logging"
	"github.com/punt-labs/punt-observe-go/telemetry"
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
	observe *observability
	json    bool
	quiet   bool
}

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(args []string, stdout, stderr io.Writer) int {
	call, parseErr := parseInvocation(args)
	cmd := command{stdout: stdout, stderr: stderr}
	if parseErr == nil && call.help {
		if err := cmd.runInvocation(call, nil); err != nil {
			fmt.Fprintln(stderr, err)
			return exitUsage
		}
		return 0
	}
	handler, err := logging.New(toolName, "cli")
	var sink slog.Handler = slog.DiscardHandler
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		if !call.quiet {
			fmt.Fprintln(stderr, "circuit: private diagnostics unavailable: timed out waiting for the rotation lock")
		}
	case err != nil:
		if !call.quiet {
			fmt.Fprintln(stderr, "circuit: private diagnostics unavailable")
		}
	default:
		sink = handler
		defer func() {
			if err := handler.Close(); err != nil && !call.quiet {
				fmt.Fprintln(stderr, "circuit: could not close private diagnostics")
			}
		}()
	}
	logger := slog.New(sink).With("component", "circuit.cli")
	usage := telemetry.New(toolName, telemetry.Config{
		Disabled:    telemetryDisabled(logger),
		Diagnostics: logger.With("component", "circuit.telemetry"),
	}, commandSchema())
	defer usage.Shutdown(context.Background())

	cmd.observe = &observability{logger: logger, usage: usage}
	if err := cmd.runInvocation(call, parseErr); err != nil {
		if !call.json {
			fmt.Fprintln(stderr, err)
		}
		return exitUsage
	}
	return 0
}

func (cmd command) dispatch(call invocation) (bool, error) {
	switch call.action {
	case actionMachineList:
		return false, cmd.listMachines()
	case actionMachineCheck:
		return false, cmd.checkMachine(call.args[0])
	case actionScaffold:
		return false, cmd.scaffold(call.args[0])
	case actionStart:
		return false, cmd.start(call.args[0])
	case actionSessionList:
		return false, cmd.listSessions()
	case actionSessionShow:
		return false, cmd.showSession(call.args[0])
	case actionAdvance:
		return cmd.advance(call.args[0], optionalArg(call.args[1:]))
	case actionStop:
		return false, cmd.stop(optionalArg(call.args))
	case actionRemove:
		return false, cmd.remove(call.args[0])
	case actionDrive:
		return false, cmd.drive(call.args[0], call.task)
	default:
		return false, cmd.dispatchRoot(call)
	}
}

func (cmd command) dispatchRoot(call invocation) error {
	switch call.action {
	case actionStatus:
		return cmd.status()
	case actionVersion:
		return cmd.version()
	case actionDoctor:
		return cmd.doctor()
	case actionInstall:
		return cmd.install()
	case actionCompletion:
		return cmd.completion(call.args[0])
	default:
		return fmt.Errorf("unknown command: %s", call.action)
	}
}

func optionalArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func (cmd command) listMachines() error {
	runtime := circuitrun.New(cmd.workingDir())
	names, err := runtime.ListMachines()
	if err != nil {
		return err
	}
	if cmd.json {
		return cmd.writeJSON(names)
	}
	for _, name := range names {
		fmt.Fprintln(cmd.stdout, name)
	}
	return nil
}

func (cmd command) checkMachine(machine string) error {
	runtime := circuitrun.New(cmd.workingDir())
	report, err := runtime.Load(machine)
	if err != nil {
		return err
	}
	if cmd.json {
		return cmd.writeJSON(machineCheckJSON{Machine: report.MachineName, Checks: report.Checks})
	}
	fmt.Fprintf(cmd.stdout, "checked: %s\n", report.MachineName)
	cmd.printCheckBindings(report.Checks)
	return nil
}

func (cmd command) scaffold(machine string) error {
	runtime := circuitrun.New(cmd.workingDir())
	report, err := runtime.Scaffold(machine)
	if err != nil {
		return err
	}
	if cmd.json {
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

func (cmd command) start(machine string) error {
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
	if cmd.json {
		return cmd.writeStatusReportJSON(report)
	}
	fmt.Fprintf(cmd.stdout, "started: %s\n", report.MachineName)
	fmt.Fprintf(cmd.stdout, "session: %s\n", id)
	cmd.printStatusReport(report)
	return nil
}

func (cmd command) drive(machine, task string) error {
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
	defer cmd.closeDriveTrace(trace)
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
	lock, err := cmd.lockSessions(syscall.LOCK_EX)
	if err != nil {
		return nil, "", nil, err
	}
	defer cmd.closeSessionLock(lock)
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return nil, "", nil, err
	}
	id, report, err := runtime.Start(machine)
	if err != nil {
		return nil, "", nil, err
	}
	if !cmd.json && !cmd.quiet {
		fmt.Fprintf(cmd.stdout, "started: %s\n", report.MachineName)
		fmt.Fprintf(cmd.stdout, "session: %s\n", id)
	}
	trace, err := cmd.openDriveTrace(runtime, id)
	return runtime, id, trace, err
}

func (cmd command) printDriveTransitions(trace io.Writer, transitions []circuitrun.AdvanceReport) {
	for _, transition := range transitions {
		writeDriveTransition(trace, transition)
		if transition.Allowed && !cmd.json && !cmd.quiet {
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
	lock, err := cmd.lockSessions(syscall.LOCK_EX)
	if err != nil {
		return err
	}
	defer cmd.closeSessionLock(lock)
	if err := runtime.PersistNewSession(id); err != nil {
		return err
	}
	if cmd.json {
		return cmd.writeStatusReportJSON(status)
	}
	if cmd.quiet {
		fmt.Fprintf(cmd.stdout, "session: %s\n", id)
	}
	fmt.Fprintf(cmd.stdout, "terminal: %s\n", status.Current)
	return nil
}

type driveTrace interface {
	io.Writer
	Close() error
}

func (cmd command) closeDriveTrace(trace driveTrace) {
	if err := trace.Close(); err != nil {
		cmd.diagnostics().Warn("Could not close the drive trace")
		if !cmd.quiet {
			fmt.Fprintln(cmd.stderr, "circuit: could not close the drive trace; it may be incomplete")
		}
	}
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

func (cmd command) openDriveTrace(runtime *circuitrun.Runtime, sessionID string) (driveTrace, error) {
	path, err := runtime.PrepareDriveTrace(sessionID)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil { //nolint:gosec // G703: path is session-scoped in the private store
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) //nolint:gosec // G304: path is a session-scoped trace in the private store
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
	return launchPiBackend(cmd.workingDir(), cmd.diagnostics().With("component", "circuit.pi"))
}

func launchPiBackend(cwd string, logger *slog.Logger) (circuitrpc.AgentBackend, func(), error) {
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
	cleanup := func() { stopPiBackend(stdin, process.Process.Signal, process.Wait, logger) }
	return backend, cleanup, nil
}

func stopPiBackend(stdin io.Closer, signal func(os.Signal) error, wait func() error, logger *slog.Logger) {
	if err := stdin.Close(); err != nil {
		logger.Warn("Could not close the agent input pipe")
	}
	signalErr := signal(os.Interrupt)
	if signalErr != nil && !errors.Is(signalErr, os.ErrProcessDone) {
		logger.Warn("Could not interrupt the agent process")
	}
	if err := wait(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			logger.Warn("Could not wait for the agent process")
			return
		}
		if signalErr == nil {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signal() == syscall.SIGINT {
				return
			}
		}
		logger.Warn("Agent process exited unexpectedly", "exit_code", exitErr.ExitCode())
	}
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

func (cmd command) status() error {
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	active, stopped := runtime.SessionCounts()
	if cmd.json {
		return cmd.writeJSON(map[string]int{"active": active, "stopped": stopped})
	}
	fmt.Fprintf(cmd.stdout, "sessions: %d active, %d stopped\n", active, stopped)
	return nil
}

func (cmd command) showSession(session string) error {
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	report, err := runtime.StatusByID(session)
	if err != nil {
		return err
	}
	if cmd.json {
		return cmd.writeStatusReportJSON(report)
	}
	if err := runtime.Suspend(); err != nil {
		return err
	}
	cmd.printStatusReport(report)
	return nil
}

func (cmd command) listSessions() error {
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	reports, err := runtime.StatusAll()
	if err != nil {
		return err
	}
	if cmd.json {
		return cmd.writeStatusJSON(reports)
	}
	if len(reports) == 0 {
		fmt.Fprintln(cmd.stdout, "no session")
		return nil
	}
	if err := runtime.Suspend(); err != nil {
		return err
	}
	cmd.printStatusReports(reports)
	return nil
}

type machineCheckJSON struct {
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

type removeJSONEntry struct {
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
	encoder := json.NewEncoder(cmd.stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entry)
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

func (cmd command) advance(event, session string) (bool, error) {
	pending, err := cmd.prepareAdvance(session, event)
	if err != nil {
		return false, err
	}
	report, err := pending.Execute()
	if err != nil {
		return false, err
	}
	if err := cmd.commitAdvance(pending); err != nil {
		return false, err
	}
	if cmd.json {
		return !report.Allowed, cmd.writeAdvanceJSON(report)
	}
	if !report.Allowed {
		fmt.Fprintf(cmd.stdout, "blocked: Advance(%s)\n", report.Event)
		for _, failed := range report.Failed {
			fmt.Fprintf(cmd.stdout, "  needs: %s\n", failed)
		}
		cmd.printChecks(report.Checks)
		return true, nil
	}
	fmt.Fprintf(cmd.stdout, "advanced: %s -> %s\n", report.From, report.To)
	cmd.printChecks(report.Checks)
	return false, nil
}

func (cmd command) remove(session string) error {
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	if err := runtime.UnloadByID(session); err != nil {
		return err
	}
	if cmd.json {
		return cmd.writeJSON(removeJSONEntry{Session: session, SessionState: string(circuitrun.SessionUnloaded)})
	}
	fmt.Fprintln(cmd.stdout, "removed")
	return nil
}

func (cmd command) stop(session string) error {
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	if session != "" {
		err = runtime.StopByID(session)
	} else {
		session, err = runtime.StopSession()
	}
	if err != nil {
		return err
	}
	if cmd.json {
		report, err := runtime.StatusByID(session)
		if err != nil {
			return err
		}
		return cmd.writeStatusReportJSON(report)
	}
	fmt.Fprintln(cmd.stdout, "stopped")
	return nil
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

func (cmd command) workingDir() string {
	if cmd.cwd != "" {
		return cmd.cwd
	}
	return "."
}
