package main

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/punt-labs/punt-observe-go/telemetry"
)

const (
	toolName       = "circuit"
	commandEvent   = "circuit.cli.command"
	commandVersion = 1

	actionHelp    = "help"
	actionStatus  = "status"
	actionAdvance = "advance"
	actionVersion = "version"
	actionMissing = "missing"
	actionUnknown = "unknown"
)

type actionName telemetry.EnumMember

type commandOutcome telemetry.EnumMember

const (
	commandSucceeded commandOutcome = "succeeded"
	commandFailed    commandOutcome = "failed"
	commandBlocked   commandOutcome = "blocked"
)

var cliActions = []telemetry.EnumMember{
	"list", "load", "scaffold", "start", actionStatus, actionAdvance, "stop", "unload", "drive", actionVersion,
	actionHelp,
}

func commandAction(args []string) actionName {
	if len(args) == 0 {
		return actionMissing
	}
	switch args[0] {
	case "--help", "-h":
		return actionHelp
	}
	if slices.Contains(cliActions, telemetry.EnumMember(args[0])) {
		return actionName(args[0])
	}
	return actionUnknown
}

type observability struct {
	logger *slog.Logger
	usage  *telemetry.Client
}

func telemetryDisabled(logger *slog.Logger) bool {
	home, err := os.UserHomeDir()
	if err == nil && filepath.IsAbs(home) {
		_, err = os.Lstat(filepath.Join(home, ".punt-labs", toolName, "telemetry-off"))
		if err == nil || errors.Is(err, fs.ErrNotExist) {
			return err == nil
		}
	}
	logger.Warn("Could not check the telemetry opt-out; capture disabled")
	return true
}

func commandSchema() telemetry.Schema {
	actions := append(slices.Clone(cliActions), actionMissing, actionUnknown)
	return telemetry.Schema{
		Name:    commandEvent,
		Version: commandVersion,
		Signal:  telemetry.Event,
		Fields: map[string]telemetry.Field{
			"action": {Kind: telemetry.Enum, Required: true, Members: actions},
			"outcome": {Kind: telemetry.Enum, Required: true, Members: []telemetry.EnumMember{
				telemetry.EnumMember(commandSucceeded), telemetry.EnumMember(commandFailed), telemetry.EnumMember(commandBlocked),
			}},
			"duration_ms": {Kind: telemetry.Duration, Required: true},
		},
	}
}

func (o *observability) record(action actionName, blocked bool, err error, elapsed time.Duration) {
	outcome := commandSucceeded
	switch {
	case err != nil:
		outcome = commandFailed
	case blocked:
		outcome = commandBlocked
	}
	if errors.Is(err, fs.ErrPermission) {
		o.logger.Warn("Command was denied filesystem access", "action", string(action))
	}
	o.logger.Info("Command finished", "action", string(action), "outcome", string(outcome))
	o.usage.Record(telemetry.Observation{
		Name:    commandEvent,
		Version: commandVersion,
		Attributes: []telemetry.Value{
			telemetry.EnumValue("action", telemetry.EnumMember(action)),
			telemetry.EnumValue("outcome", telemetry.EnumMember(outcome)),
			telemetry.Elapsed("duration_ms", elapsed),
		},
	})
}

func (cmd command) diagnostics() *slog.Logger {
	if cmd.observe == nil {
		return slog.New(slog.DiscardHandler)
	}
	return cmd.observe.logger
}
