package main

import (
	"bytes"
	"encoding/json"
	"runtime/debug"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	cases := []struct {
		name    string
		ldflags string
		info    *debug.BuildInfo
		ok      bool
		want    string
	}{
		{name: "ldflags", ldflags: "1.2.3", want: "1.2.3"},
		{name: "module version", info: &debug.BuildInfo{Main: debug.Module{Version: "v0.4.0"}}, ok: true, want: "v0.4.0"},
		{name: "devel build", info: &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, ok: true, want: "dev"},
		{name: "no build info", want: "dev"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			origVersion, origRead := version, readBuildInfo
			t.Cleanup(func() { version, readBuildInfo = origVersion, origRead })
			version = tc.ldflags
			readBuildInfo = func() (*debug.BuildInfo, bool) { return tc.info, tc.ok }
			if got := resolveVersion(); got != tc.want {
				t.Fatalf("resolveVersion() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestVersionCommand(t *testing.T) {
	origVersion := version
	t.Cleanup(func() { version = origVersion })
	version = "9.8.7"

	stdout := &bytes.Buffer{}
	cmd := command{stdout: stdout, stderr: &bytes.Buffer{}}
	if err := cmd.run([]string{"version"}); err != nil {
		t.Fatalf("version: %v", err)
	}
	if got, want := stdout.String(), "circuit 9.8.7\n"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}

	stdout.Reset()
	if err := cmd.run([]string{"version", "--json"}); err != nil {
		t.Fatalf("version --json: %v", err)
	}
	var payload map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode %q: %v", stdout.String(), err)
	}
	if payload["version"] != "9.8.7" {
		t.Fatalf("version --json = %v, want version 9.8.7", payload)
	}

	if err := cmd.run([]string{"version", "extra"}); err == nil {
		t.Fatal("version accepted an unexpected argument")
	}
}
