package main

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// version is set at build time with -ldflags "-X main.version=$(VERSION)"
// by the Makefile's build and dist targets. It stays empty for a bare
// `go build` or `go install`, and resolveVersion then falls back to the
// module version recorded in the build info.
var version string

// readBuildInfo lets tests replace debug.ReadBuildInfo.
var readBuildInfo = debug.ReadBuildInfo

func resolveVersion() string {
	if version != "" {
		return version
	}
	if info, ok := readBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "0.0.0-dev"
}

func (cmd command) version() error {
	v := resolveVersion()
	if cmd.json {
		return cmd.writeJSON(map[string]string{"version": v})
	}
	if _, err := fmt.Fprintf(cmd.stdout, "circuit %s\n", v); err != nil {
		return fmt.Errorf("write version: %w", err)
	}
	return nil
}
