package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/punt-labs/circuit/internal/circuitpath"
)

type doctorCheck struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	OK       bool   `json:"ok"`
	Detail   string `json:"detail"`
}

type doctorReport struct {
	OK      bool          `json:"ok"`
	Checks  []doctorCheck `json:"checks"`
	Failure string        `json:"error,omitempty"`
}

func (report *doctorReport) Error() string {
	return report.Failure
}

func (cmd command) doctor() error {
	checks := []doctorCheck{
		{Name: toolName, Required: true, OK: true, Detail: resolveVersion()},
		checkCircuitHome(),
		checkOptionalCommand("pi", "needed for session drive"),
		checkOptionalCommand("git", "needed for linked worktree discovery"),
	}
	report := doctorReport{OK: true, Checks: checks}
	for _, check := range checks {
		if check.Required && !check.OK {
			report.OK = false
		}
		if !cmd.json {
			result := "pass"
			if !check.OK {
				result = "fail"
			}
			kind := "optional"
			if check.Required {
				kind = "required"
			}
			if _, err := fmt.Fprintf(cmd.stdout, "%s: %s (%s) — %s\n", result, check.Name, kind, check.Detail); err != nil {
				return err
			}
		}
	}
	if !report.OK {
		report.Failure = "required Circuit installation checks failed"
		return &report
	}
	if cmd.json {
		return cmd.writeJSON(report)
	}
	return nil
}

func checkCircuitHome() doctorCheck {
	check := doctorCheck{Name: "home", Required: true}
	path, err := circuitpath.HomeRoot()
	if err == nil {
		var info os.FileInfo
		info, err = os.Stat(filepath.Dir(filepath.Dir(path)))
		if err == nil && !info.IsDir() {
			err = fmt.Errorf("HOME is not a directory")
		}
	}
	if err != nil {
		check.Detail = err.Error()
		return check
	}
	check.OK, check.Detail = true, path
	return check
}

func checkOptionalCommand(name, purpose string) doctorCheck {
	check := doctorCheck{Name: name}
	path, err := exec.LookPath(name)
	if err != nil {
		check.Detail = "not found on PATH; " + purpose
		return check
	}
	check.OK, check.Detail = true, path
	return check
}

func (cmd command) install() error {
	root, err := circuitpath.HomeRoot()
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find Circuit executable: %w", err)
	}
	destination := filepath.Join(filepath.Dir(filepath.Dir(root)), ".local", "bin", toolName)
	if err := copyExecutable(executable, destination); err != nil {
		return fmt.Errorf("install Circuit: %w", err)
	}
	if cmd.json {
		return cmd.writeJSON(map[string]string{"path": destination})
	}
	_, err = fmt.Fprintln(cmd.stdout, "installed:", destination)
	return err
}

func copyExecutable(executable, destination string) error {
	src, err := os.Open(executable) //nolint:gosec // G304: source is the running executable
	if err != nil {
		return err
	}
	defer src.Close() //nolint:errcheck // the executable is opened read-only
	dir := filepath.Dir(destination)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	dst, err := os.CreateTemp(dir, ".circuit-*")
	if err != nil {
		return err
	}
	err = writeExecutable(dst, src)
	if err == nil {
		err = os.Rename(dst.Name(), destination)
	}
	if err != nil {
		return errors.Join(err, os.Remove(dst.Name()))
	}
	return nil
}

func writeExecutable(dst *os.File, src io.Reader) error {
	if _, err := io.Copy(dst, src); err != nil {
		return errors.Join(err, dst.Close())
	}
	if err := dst.Chmod(0o755); err != nil {
		return errors.Join(err, dst.Close())
	}
	return dst.Close()
}
