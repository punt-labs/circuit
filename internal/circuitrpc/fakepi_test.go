package circuitrpc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/punt-labs/circuit/internal/circuitrun"
)

func TestRunnerLoopAgainstFakePi(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := circuitrun.Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	_, status, err := runtime.Start("build-job")
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	responses := map[string]string{
		"idle":    "start",
		"running": "finish",
	}

	piIn, runnerOut := io.Pipe()
	runnerIn, piOut := io.Pipe()

	go fakePi(t, bufio.NewReader(piIn), piOut, responses)

	writer := bufio.NewWriter(runnerOut)
	reader := bufio.NewReader(runnerIn)

	transitions := []string{}

	for {
		if IsTerminal(status) {
			break
		}

		prompt := FormatPrompt(status)
		request := map[string]string{
			"id":      "req",
			"type":    "prompt",
			"message": prompt,
		}
		data, _ := json.Marshal(request)
		_, err := fmt.Fprintf(writer, "%s\n", data)
		if err != nil {
			t.Fatalf("write prompt: %v", err)
		}
		writer.Flush()

		var lastText string
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				t.Fatalf("read event: %v", err)
			}
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				continue
			}
			if event["type"] == "message_end" {
				if msg, ok := event["message"].(map[string]interface{}); ok {
					lastText = ExtractTextFromMessage(msg)
				}
			}
			if event["type"] == "agent_settled" {
				break
			}
		}

		operation := ExtractOperation(lastText, status)
		if operation == "" {
			t.Fatalf("no operation extracted from %q", lastText)
		}

		report, err := runtime.Advance(operation)
		if err != nil {
			t.Fatalf("advance %s: %v", operation, err)
		}
		if !report.Allowed {
			t.Fatalf("advance %s blocked", operation)
		}
		transitions = append(transitions, fmt.Sprintf("%s -> %s", report.From, report.To))

		if !runtime.IsActive() {
			break
		}

		status, err = runtime.Status()
		if err != nil {
			t.Fatalf("status: %v", err)
		}
	}

	piIn.Close()
	piOut.Close()

	if len(transitions) != 2 {
		t.Fatalf("transitions = %v, want 2", transitions)
	}
	if transitions[0] != "idle -> running" {
		t.Fatalf("transition 0 = %s", transitions[0])
	}
	if transitions[1] != "running -> done" {
		t.Fatalf("transition 1 = %s", transitions[1])
	}
}

func fakePi(t *testing.T, reader *bufio.Reader, writer io.Writer, responses map[string]string) {
	t.Helper()
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		var request map[string]interface{}
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			continue
		}
		if request["type"] != "prompt" {
			continue
		}
		message, _ := request["message"].(string)
		response := ""
		for keyword, reply := range responses {
			if strings.Contains(message, "Current state: "+keyword) {
				response = reply
				break
			}
		}

		msgEnd := map[string]interface{}{
			"type": "message_end",
			"message": map[string]interface{}{
				"role": "assistant",
				"content": []interface{}{
					map[string]interface{}{"type": "text", "text": response},
				},
			},
		}
		data, _ := json.Marshal(msgEnd)
		fmt.Fprintf(writer, "%s\n", data)

		settled := map[string]string{"type": "agent_settled"}
		data, _ = json.Marshal(settled)
		fmt.Fprintf(writer, "%s\n", data)
	}
}

func testRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	machines := filepath.Join(root, "machines")
	if err := os.MkdirAll(machines, 0o700); err != nil {
		t.Fatalf("create machines dir: %v", err)
	}
	for _, name := range []string{"build-job.mch"} {
		content, err := os.ReadFile(filepath.Join("..", "..", "machines", name))
		if err != nil {
			t.Fatalf("read fixture %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(machines, name), content, 0o600); err != nil {
			t.Fatalf("write fixture %s: %v", name, err)
		}
	}
	return root
}
