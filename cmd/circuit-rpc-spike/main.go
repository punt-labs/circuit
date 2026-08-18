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

	"github.com/punt-labs/circuit/internal/circuitrpc"
	"github.com/punt-labs/circuit/internal/circuitrun"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "circuit-rpc-spike: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	machine := selectedMachine(os.Args)
	runtime, err := startRuntime(machine)
	if err != nil {
		return err
	}
	pi, err := startPi()
	if err != nil {
		return fmt.Errorf("start pi: %w", err)
	}
	defer pi.stop()
	if err := runLoop(runtime, pi); err != nil {
		return err
	}
	if err := runtime.Stop(); err != nil {
		return fmt.Errorf("stop runtime: %w", err)
	}
	return nil
}

func selectedMachine(args []string) string {
	if len(args) > 1 {
		return args[1]
	}
	return "build-job"
}

func startRuntime(machine string) (*circuitrun.Runtime, error) {
	runtime, err := circuitrun.Resume(".")
	if err != nil {
		return nil, fmt.Errorf("resume runtime: %w", err)
	}
	_, status, err := runtime.Start(machine)
	if err != nil {
		return nil, fmt.Errorf("start machine %s: %w", machine, err)
	}
	fmt.Printf("started: %s (current: %s)\n", machine, status.Current)
	return runtime, nil
}

func runLoop(runtime *circuitrun.Runtime, pi *piRPC) error {
	for {
		status, err := runtime.Status()
		if err != nil {
			return fmt.Errorf("status: %w", err)
		}
		if circuitrpc.IsTerminal(status) {
			fmt.Printf("terminal: %s\n", status.Current)
			return nil
		}
		if shouldStop, err := runStep(runtime, pi, status); err != nil || shouldStop {
			return err
		}
	}
}

func runStep(runtime *circuitrun.Runtime, pi *piRPC, status circuitrun.StatusReport) (bool, error) {
	prompt := circuitrpc.FormatPrompt(status)
	fmt.Printf("prompt: %s\n", truncate(prompt, 120))
	response, err := pi.prompt(prompt)
	if err != nil {
		return false, fmt.Errorf("pi prompt: %w", err)
	}
	fmt.Printf("response: %s\n", truncate(response, 200))
	operation := circuitrpc.ExtractOperation(response, status)
	if operation == "" {
		fmt.Println("no valid operation extracted from response")
		return true, nil
	}
	return advanceRuntime(runtime, operation)
}

func advanceRuntime(runtime *circuitrun.Runtime, operation string) (bool, error) {
	fmt.Printf("extracted operation: %s\n", operation)
	report, err := runtime.Advance(operation)
	if err != nil {
		return false, fmt.Errorf("advance %s: %w", operation, err)
	}
	if !report.Allowed {
		fmt.Printf("blocked: Advance(%s)\n", operation)
		for _, failed := range report.Failed {
			fmt.Printf("  needs: %s\n", failed)
		}
		return true, nil
	}
	fmt.Printf("advanced: %s -> %s\n", report.From, report.To)
	if err := runtime.Suspend(); err != nil {
		return false, fmt.Errorf("suspend: %w", err)
	}
	return false, nil
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
	request := circuitrpc.PromptRequest{
		ID:      fmt.Sprintf("req-%d", time.Now().UnixNano()),
		Type:    "prompt",
		Message: message,
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
		var event circuitrpc.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		if event.Type == "message_end" && event.Message.Role == "assistant" {
			lastAssistantText = circuitrpc.ExtractTextFromMessage(event.Message)
		}
		if event.Type == "agent_settled" {
			return lastAssistantText, nil
		}
	}

	return lastAssistantText, fmt.Errorf("timed out waiting for agent_settled")
}

func (p *piRPC) stop() {
	if err := p.stdin.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "close stdin: %v\n", err)
	}
	if err := p.cmd.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "wait process: %v\n", err)
	}
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
