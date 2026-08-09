package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/punt-labs/circuit/internal/circuitrun"
)

const exitUsage = 2

type command struct {
	stdout io.Writer
	stderr io.Writer
	cwd    string
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
	machine, err := singleArg(args)
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
	fmt.Fprintf(cmd.stdout, "loaded: %s\n", report.MachineName)
	cmd.printCheckBindings(report.Checks)
	return nil
}

func (cmd command) scaffold(args []string) error {
	machine, err := singleArg(args)
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
	fmt.Fprintf(cmd.stdout, "scaffolded: %s\n", report.MachineName)
	for _, variable := range report.GeneratedBindings {
		fmt.Fprintf(cmd.stdout, "binding: %s\n", variable)
	}
	for _, check := range report.GeneratedRegistryIDs {
		fmt.Fprintf(cmd.stdout, "stub: %s -> false\n", check)
	}
	return nil
}

func (cmd command) start(args []string) error {
	machine, err := singleArg(args)
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
	fmt.Fprintf(cmd.stdout, "started: %s\n", report.MachineName)
	fmt.Fprintf(cmd.stdout, "session: %s\n", id)
	cmd.printStatusReport(report)
	return nil
}

func (cmd command) status(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("expected at most one argument, got %d", len(args))
	}
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	if !runtime.IsActive() {
		fmt.Fprintln(cmd.stdout, "no active session")
		return nil
	}
	if len(args) == 1 {
		report, err := runtime.StatusByID(args[0])
		if err != nil {
			return err
		}
		cmd.printStatusReport(report)
		return runtime.Suspend()
	}
	reports, err := runtime.StatusAll()
	if err != nil {
		return err
	}
	cmd.printStatusReports(reports)
	return runtime.Suspend()
}

func (cmd command) advance(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("expected one event and optional session, got %d arguments", len(args))
	}
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	var report circuitrun.AdvanceReport
	if len(args) == 2 {
		report, err = runtime.AdvanceByID(args[1], args[0])
	} else {
		report, err = runtime.Advance(args[0])
	}
	if err != nil {
		return err
	}
	if err := runtime.Suspend(); err != nil {
		return err
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

func (cmd command) stop(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("expected at most one argument, got %d", len(args))
	}
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	if len(args) == 1 {
		err = runtime.StopByID(args[0])
	} else {
		err = runtime.Stop()
	}
	if err != nil {
		return err
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
	fmt.Fprintln(cmd.stderr, "  load <machine>              validate machine and check bindings")
	fmt.Fprintln(cmd.stderr, "  scaffold <machine>          generate missing check bindings and false stubs")
	fmt.Fprintln(cmd.stderr, "  start <machine>             start a circuit session")
	fmt.Fprintln(cmd.stderr, "  status [session]            print circuit session status")
	fmt.Fprintln(cmd.stderr, "  advance <event> [session]   apply Advance(event) to a circuit session")
	fmt.Fprintln(cmd.stderr, "  stop [session]              clear a circuit session")
}

func (cmd command) workingDir() string {
	if cmd.cwd != "" {
		return cmd.cwd
	}
	return "."
}

func singleArg(args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("expected exactly one argument, got %d", len(args))
	}
	return args[0], nil
}
