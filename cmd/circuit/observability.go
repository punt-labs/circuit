package main

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/punt-labs/circuit/internal/circuitpath"
	"github.com/punt-labs/punt-observe-go/telemetry"
)

const (
	toolName       = circuitpath.ToolName
	commandEvent   = "circuit.cli.command"
	commandVersion = 2
)

type actionName telemetry.EnumMember

type commandOutcome telemetry.EnumMember

const (
	commandSucceeded commandOutcome = "succeeded"
	commandFailed    commandOutcome = "failed"
	commandBlocked   commandOutcome = "blocked"
)

type observability struct {
	logger *slog.Logger
	usage  *telemetry.Client
}

func telemetryDisabled(logger *slog.Logger) bool {
	home, err := circuitpath.HomeRoot()
	if err == nil {
		_, err = os.Lstat(filepath.Join(home, "telemetry-off"))
		if err == nil || errors.Is(err, fs.ErrNotExist) {
			return err == nil
		}
	}
	logger.Warn("Could not check the telemetry opt-out; capture disabled")
	return true
}

func commandSchema() telemetry.Schema {
	actions := make([]telemetry.EnumMember, 0, 2+len(commandTree))
	actions = append(actions, telemetry.EnumMember(actionHelp), telemetry.EnumMember(actionUnknown))
	for _, spec := range commandTree {
		actions = append(actions, commandActionMember(spec.action))
	}
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

func commandActionMember(action actionName) telemetry.EnumMember {
	return telemetry.EnumMember(strings.ReplaceAll(string(action), " ", "_"))
}

func outcomeFor(blocked bool, err error) commandOutcome {
	if err != nil {
		return commandFailed
	}
	if blocked {
		return commandBlocked
	}
	return commandSucceeded
}

func (o *observability) record(action actionName, blocked bool, err error, elapsed time.Duration) {
	outcome := outcomeFor(blocked, err)
	if errors.Is(err, fs.ErrPermission) {
		o.logger.Warn("Command was denied filesystem access", "action", string(action))
	}
	o.logger.Info("Command finished", "action", string(action), "outcome", string(outcome))
	o.usage.Record(telemetry.Observation{
		Name:    commandEvent,
		Version: commandVersion,
		Attributes: []telemetry.Value{
			telemetry.EnumValue("action", commandActionMember(action)),
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
