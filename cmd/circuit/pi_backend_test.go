package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestPiBackendDiagnosticsRespectQuiet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	pi := `#!/bin/sh
printf '%s\n' 'Pi diagnostic' >&2
IFS= read -r request || exit 1
printf '%s\n' '{"type":"agent_settled"}'
IFS= read -r request
`
	if err := os.WriteFile(filepath.Join(dir, "pi"), []byte(pi), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		quiet bool
		want  string
	}{
		{name: "normal", want: "Pi diagnostic\n"},
		{name: "quiet", quiet: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stderr bytes.Buffer
			cmd := command{cwd: dir, stderr: &stderr, quiet: tc.quiet}
			backend, stop, err := cmd.agentBackend()
			if err != nil {
				t.Fatal(err)
			}
			_, err = backend.Prompt("test")
			stop()
			if err != nil || stderr.String() != tc.want {
				t.Fatalf("Pi prompt: %v; stderr = %q, want %q", err, &stderr, tc.want)
			}
		})
	}
}

func TestStopPiBackend(t *testing.T) {
	t.Parallel()

	privateErr := errors.New("private shutdown failure")
	for _, tc := range []struct {
		name      string
		closeErr  error
		signalErr error
		waitErr   error
		childExit string
		wantLog   string
	}{
		{name: "clean wait"},
		{
			name:     "stdin close failure",
			closeErr: privateErr,
			wantLog:  "level=WARN msg=\"Could not close the agent input pipe\"\n",
		},
		{name: "process already done", signalErr: os.ErrProcessDone},
		{
			name:      "interrupt failure",
			signalErr: privateErr,
			wantLog:   "level=WARN msg=\"Could not interrupt the agent process\"\n",
		},
		{name: "expected SIGINT exit", childExit: "interrupt"},
		{
			name:      "unexpected nonzero exit",
			signalErr: os.ErrProcessDone,
			childExit: "nonzero",
			wantLog:   "level=WARN msg=\"Agent process exited unexpectedly\" exit_code=7\n",
		},
		{
			name:      "nonzero exit after successful interrupt",
			childExit: "nonzero",
			wantLog:   "level=WARN msg=\"Agent process exited unexpectedly\" exit_code=7\n",
		},
		{
			name:    "non-exit wait failure",
			waitErr: privateErr,
			wantLog: "level=WARN msg=\"Could not wait for the agent process\"\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			waitErr := tc.waitErr
			if tc.childExit != "" {
				waitErr = piBackendExitError(t, tc.childExit)
			}
			var logs bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{
				ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
					if attr.Key == slog.TimeKey {
						return slog.Attr{}
					}
					return attr
				},
			}))
			var calls []string
			stopPiBackend(piInputCloser(func() error {
				calls = append(calls, "close")
				return tc.closeErr
			}), func(sig os.Signal) error {
				calls = append(calls, "signal")
				if sig != os.Interrupt {
					t.Fatalf("signal = %v, want %v", sig, os.Interrupt)
				}
				return tc.signalErr
			}, func() error {
				calls = append(calls, "wait")
				return waitErr
			}, logger)
			if want := []string{"close", "signal", "wait"}; !slices.Equal(calls, want) {
				t.Errorf("shutdown calls = %v, want %v", calls, want)
			}
			if got := logs.String(); got != tc.wantLog {
				t.Errorf("logs = %q, want %q", got, tc.wantLog)
			}
		})
	}
}

type piInputCloser func() error

func (c piInputCloser) Close() error { return c() }

func piBackendExitError(t *testing.T, outcome string) *exec.ExitError {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPiBackendExitHelper$")
	cmd.Env = append(os.Environ(), "CIRCUIT_TEST_PI_EXIT="+outcome)
	err := cmd.Run()
	var exitErr *exec.ExitError
	if ctx.Err() != nil || !errors.As(err, &exitErr) {
		t.Fatalf("child did not exit as requested: %v (context: %v)", err, ctx.Err())
	}
	return exitErr
}

func TestPiBackendExitHelper(t *testing.T) {
	switch os.Getenv("CIRCUIT_TEST_PI_EXIT") {
	case "nonzero":
		os.Exit(7)
	case "interrupt":
		process, err := os.FindProcess(os.Getpid())
		if err != nil {
			t.Fatal(err)
		}
		if err := process.Signal(os.Interrupt); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Minute)
	}
}
