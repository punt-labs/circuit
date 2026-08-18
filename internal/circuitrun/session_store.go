package circuitrun

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (runtime *Runtime) Suspend() error {
	if err := os.MkdirAll(runtime.sessionsDir(), 0o700); err != nil {
		return fmt.Errorf("create sessions directory: %w", err)
	}
	for _, id := range runtime.persistedSessionIDs() {
		if _, ok := runtime.sessions[id]; !ok {
			if err := os.Remove(runtime.sessionPath(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("remove stale session %s: %w", id, err)
			}
		}
	}
	for _, run := range runtime.sessions {
		if run.Session != SessionActive && run.Session != SessionStopped {
			continue
		}
		if err := runtime.writeSession(run); err != nil {
			return err
		}
	}
	return runtime.removeLegacySuspendedRun()
}

func (runtime *Runtime) loadLegacySuspendedRun() error {
	content, err := os.ReadFile(runtime.suspendedPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read legacy suspended run: %w", err)
	}
	var run Run
	if err := json.Unmarshal(content, &run); err != nil {
		return fmt.Errorf("parse legacy suspended run: %w", err)
	}
	if run.Session != SessionActive {
		return nil
	}
	if run.SessionID == "" {
		id, err := runtime.newSessionID(run.MachineName)
		if err != nil {
			return err
		}
		run.SessionID = id
	}
	if !isSafeSessionID(run.SessionID) {
		return fmt.Errorf("unsafe legacy session id: %s", run.SessionID)
	}
	runtime.sessions[run.SessionID] = &run
	runtime.currentID = run.SessionID
	return nil
}

func (runtime *Runtime) loadSessions() error {
	entries, err := os.ReadDir(runtime.sessionsDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read sessions directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(runtime.sessionsDir(), entry.Name())
		content, err := os.ReadFile(path) //nolint:gosec // G304: path is session JSON file under .tmp/sessions/
		if err != nil {
			return fmt.Errorf("read session file %s: %w", entry.Name(), err)
		}
		var run Run
		if err := json.Unmarshal(content, &run); err != nil {
			return fmt.Errorf("parse session file %s: %w", entry.Name(), err)
		}
		if run.Session != SessionActive && run.Session != SessionStopped {
			continue
		}
		if run.SessionID == "" {
			run.SessionID = strings.TrimSuffix(entry.Name(), ".json")
		}
		if !isSafeSessionID(run.SessionID) {
			continue
		}
		runtime.sessions[run.SessionID] = &run
	}
	return nil
}

func (runtime *Runtime) persistedSessionIDs() []string {
	entries, err := os.ReadDir(runtime.sessionsDir())
	if err != nil {
		return nil
	}
	ids := []string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		ids = append(ids, strings.TrimSuffix(entry.Name(), ".json"))
	}
	sort.Strings(ids)
	return ids
}

func (runtime *Runtime) newSessionID(machineName string) (string, error) {
	base := sessionIDMachineName(machineName)
	for range 16 {
		id, err := randomHex(2)
		if err != nil {
			return "", err
		}
		sessionID := base + "-" + id
		if _, ok := runtime.sessions[sessionID]; ok {
			continue
		}
		if _, err := os.Stat(runtime.sessionPath(sessionID)); errors.Is(err, os.ErrNotExist) {
			return sessionID, nil
		} else if err != nil {
			return "", fmt.Errorf("check session path: %w", err)
		}
	}
	return "", fmt.Errorf("could not allocate session id for %s", machineName)
}

func sessionIDMachineName(machineName string) string {
	name := strings.TrimSuffix(filepath.Base(machineName), ".mch")
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "machine"
	}
	return name
}

func isSafeSessionID(id string) bool {
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "\\") || strings.Contains(id, "..") {
		return false
	}
	return filepath.Base(id) == id
}

func randomHex(bytesCount int) (string, error) {
	buffer := make([]byte, bytesCount)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

func (runtime *Runtime) writeSession(run *Run) error {
	if err := os.MkdirAll(runtime.sessionsDir(), 0o700); err != nil {
		return fmt.Errorf("create sessions directory: %w", err)
	}
	content, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	if err := os.WriteFile(runtime.sessionPath(run.SessionID), append(content, '\n'), 0o600); err != nil {
		return fmt.Errorf("write session file: %w", err)
	}
	return nil
}

func (runtime *Runtime) sessionPath(id string) string {
	return filepath.Join(runtime.sessionsDir(), id+".json")
}

func (runtime *Runtime) sessionsDir() string {
	return filepath.Join(runtime.root, ".tmp", "sessions")
}

func (runtime *Runtime) suspendedPath() string {
	return filepath.Join(runtime.root, ".tmp", "circuit.suspended.json")
}

func (runtime *Runtime) removeLegacySuspendedRun() error {
	if err := os.Remove(runtime.suspendedPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove legacy suspended run: %w", err)
	}
	return nil
}
