package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCLIObservability(t *testing.T) {
	root := testObservabilityHome(t)
	raw := "private-user\nprivate-token=secret-value"
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"version", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("version exited with %d: %s", code, &stderr)
	}
	if !strings.HasPrefix(stdout.String(), "{") || stderr.Len() != 0 {
		t.Fatalf("unexpected version streams: stdout=%q stderr=%q", &stdout, &stderr)
	}
	if code := runCLI([]string{raw}, io.Discard, &stderr); code != exitUsage {
		t.Fatalf("unknown command exited with %d", code)
	}
	if !strings.Contains(stderr.String(), raw) {
		t.Fatal("missing CLI error")
	}

	logDir := filepath.Join(root, "logs")
	logPath := filepath.Join(logDir, "circuit.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 2 ||
		!strings.HasSuffix(lines[0], "[INFO] circuit.cli: Command finished action=version outcome=succeeded") ||
		!strings.HasSuffix(lines[1], "[INFO] circuit.cli: Command finished action=unknown outcome=failed") {
		t.Fatalf("unexpected diagnostic records: %q", data)
	}

	assertObservabilityMode(t, root, 0o700)
	assertObservabilityMode(t, logDir, 0o700)
	assertObservabilityMode(t, logPath, 0o600)
	assertObservabilityMode(t, filepath.Join(root, "telemetry.db"), 0o600)
}

func TestCLIObservabilityOptOut(t *testing.T) {
	for _, env := range []string{"PUNT_TELEMETRY", "CIRCUIT_TELEMETRY", ""} {
		name := env
		if name == "" {
			name = "config"
		}
		t.Run(name, func(t *testing.T) {
			root := testObservabilityHome(t)
			if env != "" {
				t.Setenv(env, "off")
			} else {
				if err := os.MkdirAll(root, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "telemetry-off"), nil, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if code := runCLI([]string{"version"}, io.Discard, io.Discard); code != 0 {
				t.Fatalf("version exited with %d", code)
			}
			assertObservabilityMode(t, filepath.Join(root, "logs", "circuit.log"), 0o600)
			if _, err := os.Stat(filepath.Join(root, "telemetry.db")); !os.IsNotExist(err) {
				t.Fatalf("telemetry enabled with %s opt-out: %v", name, err)
			}
		})
	}
}

func TestCLIContinuesWhenDiagnosticRotationLockTimesOut(t *testing.T) {
	root := testObservabilityHome(t)
	logDir := filepath.Join(root, "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		t.Fatal(err)
	}
	lock, err := os.OpenFile(filepath.Join(logDir, "circuit.log.rotate.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	watchdog := time.AfterFunc(5*time.Second, func() { _ = lock.Close() })
	defer watchdog.Stop()

	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"version", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("version exited with %d: %s", code, &stderr)
	}
	if !strings.HasPrefix(stdout.String(), "{") {
		t.Fatalf("version did not print JSON: %q", &stdout)
	}
	if stderr.String() != "circuit: private diagnostics unavailable: timed out waiting for the rotation lock\n" {
		t.Fatalf("unexpected diagnostic: %q", &stderr)
	}
	if _, err := os.Stat(filepath.Join(logDir, "circuit.log")); !os.IsNotExist(err) {
		t.Fatalf("unexpected durable log while the lock was held: %v", err)
	}
	assertObservabilityMode(t, filepath.Join(root, "telemetry.db"), 0o600)
}

func TestCLIReportsUnavailableDiagnosticsWithoutCheckoutFallback(t *testing.T) {
	root := testObservabilityHome(t)
	if err := os.MkdirAll(filepath.Dir(root), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	cwd := t.TempDir()
	t.Chdir(cwd)
	var stderr bytes.Buffer
	if code := runCLI([]string{"version"}, io.Discard, &stderr); code != 0 {
		t.Fatalf("version exited with %d", code)
	}
	if stderr.String() != "circuit: private diagnostics unavailable\n" {
		t.Fatalf("unexpected degraded diagnostic: %q", &stderr)
	}
	entries, err := os.ReadDir(cwd)
	if err != nil || len(entries) != 0 {
		t.Fatalf("unexpected checkout artifacts: %v, %v", entries, err)
	}
}

type brokenDriveTrace struct{ io.Writer }

func (brokenDriveTrace) Close() error {
	return errors.New("private contents")
}

func TestDriveTraceCloseFailureReportsIncompleteArtifactWithoutDetails(t *testing.T) {
	t.Parallel()
	var stderr bytes.Buffer
	cmd := command{stderr: &stderr}
	cmd.closeDriveTrace(brokenDriveTrace{io.Discard})
	if stderr.String() != "circuit: could not close the drive trace; it may be incomplete\n" {
		t.Fatalf("unexpected trace close diagnostic: %q", &stderr)
	}
}

func testObservabilityHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PUNT_TELEMETRY", "")
	t.Setenv("CIRCUIT_TELEMETRY", "")
	return filepath.Join(home, ".punt-labs", "circuit")
}

func assertObservabilityMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %04o, want %04o", filepath.Base(path), got, want)
	}
}
