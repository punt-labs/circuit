package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateReadableFile(t *testing.T) {
	t.Parallel()
	path := writeTempPlaybook(t)
	stdout := &bytes.Buffer{}
	cmd := command{stdout: stdout, stderr: &bytes.Buffer{}}

	err := cmd.run([]string{"validate", path})

	if err != nil {
		t.Fatalf("validate returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "valid:") {
		t.Fatalf("validate output missing valid marker: %q", stdout.String())
	}
}

func TestSummaryReadableFile(t *testing.T) {
	t.Parallel()
	path := writeTempPlaybook(t)
	stdout := &bytes.Buffer{}
	cmd := command{stdout: stdout, stderr: &bytes.Buffer{}}

	err := cmd.run([]string{"summary", path})

	if err != nil {
		t.Fatalf("summary returned error: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "file:") {
		t.Fatalf("summary output missing file line: %q", output)
	}
	if !strings.Contains(output, "size:") {
		t.Fatalf("summary output missing size line: %q", output)
	}
}

func TestMissingFileFails(t *testing.T) {
	t.Parallel()
	cmd := command{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}

	err := cmd.run([]string{"validate", filepath.Join(t.TempDir(), "missing.yaml")})

	if err == nil {
		t.Fatal("validate returned nil error for missing file")
	}
}

func TestUnknownCommandFails(t *testing.T) {
	t.Parallel()
	stderr := &bytes.Buffer{}
	cmd := command{stdout: &bytes.Buffer{}, stderr: stderr}

	err := cmd.run([]string{"nope"})

	if err == nil {
		t.Fatal("unknown command returned nil error")
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("unknown command did not print usage: %q", stderr.String())
	}
}

func writeTempPlaybook(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "playbook.yaml")
	content := []byte("name: sample\nsteps: []\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write temp playbook: %v", err)
	}
	return path
}
