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

func (cmd command) start(args []string) error {
	machine, err := singleArg(args)
	if err != nil {
		return err
	}
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	report, err := runtime.Start(machine)
	if err != nil {
		return err
	}
	if err := runtime.Suspend(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.stdout, "started: %s\n", report.MachineName)
	cmd.printStatusReport(report)
	return nil
}

func (cmd command) status(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("expected no arguments, got %d", len(args))
	}
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	if !runtime.IsActive() {
		fmt.Fprintln(cmd.stdout, "no active session")
		return nil
	}
	report, err := runtime.Status()
	if err != nil {
		return err
	}
	if err := runtime.Suspend(); err != nil {
		return err
	}
	cmd.printStatusReport(report)
	return nil
}

func (cmd command) advance(args []string) error {
	event, err := singleArg(args)
	if err != nil {
		return err
	}
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	report, err := runtime.Advance(event)
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
	if len(args) != 0 {
		return fmt.Errorf("expected no arguments, got %d", len(args))
	}
	runtime, err := circuitrun.Resume(cmd.workingDir())
	if err != nil {
		return err
	}
	if err := runtime.Stop(); err != nil {
		return err
	}
	fmt.Fprintln(cmd.stdout, "stopped")
	return nil
}

func (cmd command) printStatusReport(report circuitrun.StatusReport) {
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
	fmt.Fprintln(cmd.stderr, "  list                  list available B machines")
	fmt.Fprintln(cmd.stderr, "  start <machine>       start an active circuit")
	fmt.Fprintln(cmd.stderr, "  status                print active circuit status")
	fmt.Fprintln(cmd.stderr, "  advance <event>       apply Advance(event) to active circuit")
	fmt.Fprintln(cmd.stderr, "  stop                  clear the active circuit")
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
