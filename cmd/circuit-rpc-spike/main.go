package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/punt-labs/circuit/internal/circuitrun"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "circuit-rpc-spike: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	machine := "build-job"
	if len(os.Args) > 1 {
		machine = os.Args[1]
	}

	runtime, err := circuitrun.Resume(".")
	if err != nil {
		return fmt.Errorf("resume runtime: %w", err)
	}
	status, err := runtime.Start(machine)
	if err != nil {
		return fmt.Errorf("start machine %s: %w", machine, err)
	}
	fmt.Printf("started: %s (current: %s)\n", machine, status.Current)

	pi, err := startPi()
	if err != nil {
		return fmt.Errorf("start pi: %w", err)
	}
	defer pi.stop()

	for {
		status, err = runtime.Status()
		if err != nil {
			return fmt.Errorf("status: %w", err)
		}
		if isTerminal(status) {
			fmt.Printf("terminal: %s\n", status.Current)
			break
		}

		prompt := formatPrompt(status)
		fmt.Printf("prompt: %s\n", truncate(prompt, 120))

		response, err := pi.prompt(prompt)
		if err != nil {
			return fmt.Errorf("pi prompt: %w", err)
		}
		fmt.Printf("response: %s\n", truncate(response, 200))

		operation := extractOperation(response, status)
		if operation == "" {
			fmt.Println("no valid operation extracted from response")
			break
		}
		fmt.Printf("extracted operation: %s\n", operation)

		report, err := runtime.Advance(operation)
		if err != nil {
			return fmt.Errorf("advance %s: %w", operation, err)
		}
		if !report.Allowed {
			fmt.Printf("blocked: Advance(%s)\n", operation)
			for _, failed := range report.Failed {
				fmt.Printf("  needs: %s\n", failed)
			}
			break
		}
		fmt.Printf("advanced: %s -> %s\n", report.From, report.To)

		if err := runtime.Suspend(); err != nil {
			return fmt.Errorf("suspend: %w", err)
		}
	}

	if err := runtime.Stop(); err != nil {
		return fmt.Errorf("stop runtime: %w", err)
	}
	return nil
}

func isTerminal(status circuitrun.StatusReport) bool {
	return len(status.Enabled) == 0
}

func formatPrompt(status circuitrun.StatusReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are operating inside a Circuit state machine.\n\n")
	fmt.Fprintf(&b, "Machine: %s\n", status.MachineName)
	fmt.Fprintf(&b, "Current state: %s\n\n", status.Current)
	if len(status.Enabled) > 0 {
		fmt.Fprintf(&b, "Enabled operations:\n")
		for _, call := range status.Enabled {
			fmt.Fprintf(&b, "  %s\n", call.Call)
		}
	}
	if len(status.Blocked) > 0 {
		fmt.Fprintf(&b, "Blocked operations:\n")
		for _, call := range status.Blocked {
			fmt.Fprintf(&b, "  %s\n", call.Call)
		}
	}
	fmt.Fprintf(&b, "\nRespond with exactly one enabled operation event name, e.g.: start\n")
	fmt.Fprintf(&b, "Do not explain. Just the event name.\n")
	return b.String()
}

func extractOperation(response string, status circuitrun.StatusReport) string {
	lower := strings.ToLower(strings.TrimSpace(response))
	for _, call := range status.Enabled {
		event := extractEvent(call.Call)
		if strings.Contains(lower, strings.ToLower(event)) {
			return event
		}
	}
	return ""
}

func extractEvent(call string) string {
	start := strings.Index(call, "(")
	end := strings.Index(call, ")")
	if start >= 0 && end > start {
		return call[start+1 : end]
	}
	return call
}

type piRPC struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader
}

func startPi() (*piRPC, error) {
	cmd := exec.Command("pi", "--mode", "rpc", "--no-session", "--approve")
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &piRPC{cmd: cmd, stdin: stdin, reader: bufio.NewReader(stdout)}, nil
}

func (p *piRPC) prompt(message string) (string, error) {
	request := map[string]string{
		"id":      fmt.Sprintf("req-%d", time.Now().UnixNano()),
		"type":    "prompt",
		"message": message,
	}
	data, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	if _, err := fmt.Fprintf(p.stdin, "%s\n", data); err != nil {
		return "", err
	}

	deadline := time.Now().Add(120 * time.Second)
	var lastAssistantText string

	for time.Now().Before(deadline) {
		line, err := p.reader.ReadString('\n')
		if err != nil {
			return lastAssistantText, err
		}
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		eventType, _ := event["type"].(string)

		if eventType == "message_end" {
			if msg, ok := event["message"].(map[string]interface{}); ok {
				if role, _ := msg["role"].(string); role == "assistant" {
					lastAssistantText = extractTextFromMessage(msg)
				}
			}
		}
		if eventType == "agent_settled" {
			return lastAssistantText, nil
		}
	}

	return lastAssistantText, fmt.Errorf("timed out waiting for agent_settled")
}

func extractTextFromMessage(msg map[string]interface{}) string {
	content, ok := msg["content"].([]interface{})
	if !ok {
		return ""
	}
	var parts []string
	for _, item := range content {
		block, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if blockType, _ := block["type"].(string); blockType == "text" {
			if text, _ := block["text"].(string); text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

func (p *piRPC) stop() {
	p.stdin.Close()
	p.cmd.Wait()
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
