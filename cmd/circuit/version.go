package main

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
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
		return info.Main.Version
	}
	return "dev"
}

func (cmd command) version(args []string) error {
	jsonMode := false
	for _, arg := range args {
		if arg != jsonFlag {
			return fmt.Errorf("unexpected argument: %s", arg)
		}
		jsonMode = true
	}
	v := resolveVersion()
	if jsonMode {
		if err := json.NewEncoder(cmd.stdout).Encode(map[string]string{"version": v}); err != nil {
			return fmt.Errorf("encode version: %w", err)
		}
		return nil
	}
	if _, err := fmt.Fprintf(cmd.stdout, "circuit %s\n", v); err != nil {
		return fmt.Errorf("write version: %w", err)
	}
	return nil
}
