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

func TestRunnerLoopRejectsInvalidFakePiResponse(t *testing.T) {
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

	_, err = runFakePiStep(t, status, map[string]string{"idle": "bogus"})

	if err == nil || !strings.Contains(err.Error(), "no operation extracted") {
		t.Fatalf("invalid response error = %v, want no operation extracted", err)
	}
	statusAfter, err := runtime.Status()
	if err != nil {
		t.Fatalf("status after invalid response: %v", err)
	}
	if statusAfter.Current != "idle" {
		t.Fatalf("current after invalid response = %s, want idle", statusAfter.Current)
	}
}

func TestRunnerLoopRejectsBlockedFakePiResponse(t *testing.T) {
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

	_, err = runFakePiStep(t, status, map[string]string{"idle": "finish"})

	if err == nil || !strings.Contains(err.Error(), "no operation extracted") {
		t.Fatalf("blocked response error = %v, want no operation extracted", err)
	}
}

func TestRunnerSingleSessionOnly(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := circuitrun.Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	_, _, err = runtime.Start("build-job")
	if err != nil {
		t.Fatalf("start 1: %v", err)
	}
	_, _, err = runtime.Start("build-job")
	if err != nil {
		t.Fatalf("start 2: %v", err)
	}
	_, err = runtime.Advance("start")
	if err == nil || !strings.Contains(err.Error(), "multiple active sessions") {
		t.Fatalf("implicit advance with two sessions error = %v, want multiple active sessions", err)
	}
}

func TestRunnerPromptIncludesGoalAndCurrentStateGuidance(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := circuitrun.Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if _, _, err := runtime.Start("build-job"); err != nil {
		t.Fatalf("start: %v", err)
	}
	backend := &scriptedBackend{responses: []string{"start"}}
	guidance := DriverGuidance{
		Goal: "Prove goal and state guidance reach the agent.",
		States: map[string]StateGuidance{
			"idle": {Prompt: "Do the idle-state work before advancing.", Event: "start"},
		},
	}

	_, err = RunUntilAcceptedWithGuidance(runtime, backend, guidance)
	if err != nil {
		t.Fatalf("run until accepted: %v", err)
	}
	if len(backend.prompts) != 1 {
		t.Fatalf("prompts = %d, want 1", len(backend.prompts))
	}
	prompt := backend.prompts[0]
	for _, want := range []string{
		"Goal: Prove goal and state guidance reach the agent.",
		"Current state: idle",
		"Do the idle-state work before advancing.",
		"When ready, request transition event: start",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestRunnerRepromptsAfterBlockedOperation(t *testing.T) {
	t.Parallel()
	root := testRoot(t)
	runtime, err := circuitrun.Resume(root)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if _, _, err := runtime.Start("build-job"); err != nil {
		t.Fatalf("start: %v", err)
	}
	backend := &scriptedBackend{responses: []string{"finish", "start"}}

	result, err := RunUntilAccepted(runtime, backend)
	if err != nil {
		t.Fatalf("run until accepted: %v", err)
	}
	if result.Prompts != 2 {
		t.Fatalf("prompts = %d, want 2", result.Prompts)
	}
	if !result.Transition.Allowed || result.Transition.From != "idle" || result.Transition.To != "running" {
		t.Fatalf("transition = %#v, want idle -> running", result.Transition)
	}
	status, err := runtime.Status()
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Current != "running" {
		t.Fatalf("current = %s, want running", status.Current)
	}
}

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
		request := PromptRequest{ID: "req", Type: "prompt", Message: prompt}
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
			var event Event
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				continue
			}
			if event.Type == "message_end" {
				lastText = ExtractTextFromMessage(event.Message)
			}
			if event.Type == "agent_settled" {
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

func runFakePiStep(t *testing.T, status circuitrun.StatusReport, responses map[string]string) (string, error) {
	t.Helper()
	piIn, runnerOut := io.Pipe()
	runnerIn, piOut := io.Pipe()
	go fakePi(t, bufio.NewReader(piIn), piOut, responses)
	defer piIn.Close()
	defer piOut.Close()

	writer := bufio.NewWriter(runnerOut)
	reader := bufio.NewReader(runnerIn)
	request := PromptRequest{ID: "req", Type: "prompt", Message: FormatPrompt(status)}
	data, _ := json.Marshal(request)
	if _, err := fmt.Fprintf(writer, "%s\n", data); err != nil {
		return "", err
	}
	writer.Flush()

	var lastText string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.Type == "message_end" {
			lastText = ExtractTextFromMessage(event.Message)
		}
		if event.Type == "agent_settled" {
			break
		}
	}
	operation := ExtractOperation(lastText, status)
	if operation == "" {
		return "", fmt.Errorf("no operation extracted from %q", lastText)
	}
	return operation, nil
}

type scriptedBackend struct {
	responses []string
	prompts   []string
}

func (backend *scriptedBackend) Prompt(message string) (string, error) {
	backend.prompts = append(backend.prompts, message)
	if len(backend.responses) == 0 {
		return "", fmt.Errorf("no scripted response for prompt %d", len(backend.prompts))
	}
	response := backend.responses[0]
	backend.responses = backend.responses[1:]
	return response, nil
}

func fakePi(t *testing.T, reader *bufio.Reader, writer io.Writer, responses map[string]string) {
	t.Helper()
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		var request PromptRequest
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			continue
		}
		if request.Type != "prompt" {
			continue
		}
		message := request.Message
		response := ""
		for keyword, reply := range responses {
			if strings.Contains(message, "Current state: "+keyword) {
				response = reply
				break
			}
		}

		msgEnd := Event{
			Type: "message_end",
			Message: Message{
				Role:    "assistant",
				Content: []ContentBlock{{Type: "text", Text: response}},
			},
		}
		data, _ := json.Marshal(msgEnd)
		fmt.Fprintf(writer, "%s\n", data)

		settled := Event{Type: "agent_settled"}
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
