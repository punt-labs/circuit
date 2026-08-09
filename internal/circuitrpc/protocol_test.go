package circuitrpc

import (
	"strings"
	"testing"

	"github.com/punt-labs/circuit/internal/circuitb"
	"github.com/punt-labs/circuit/internal/circuitrun"
)

func TestFormatPromptIncludesStateAndOperations(t *testing.T) {
	t.Parallel()
	status := circuitrun.StatusReport{
		MachineName: "build-job",
		Current:     "idle",
		Enabled:     []circuitb.CallStatus{{Call: "Advance(start)"}},
		Blocked:     []circuitb.CallStatus{{Call: "Advance(finish)"}},
	}

	prompt := FormatPrompt(status)

	if !strings.Contains(prompt, "Machine: build-job") {
		t.Fatalf("prompt missing machine name: %q", prompt)
	}
	if !strings.Contains(prompt, "Current state: idle") {
		t.Fatalf("prompt missing current state: %q", prompt)
	}
	if !strings.Contains(prompt, "Advance(start)") {
		t.Fatalf("prompt missing enabled operation: %q", prompt)
	}
	if !strings.Contains(prompt, "Advance(finish)") {
		t.Fatalf("prompt missing blocked operation: %q", prompt)
	}
}

func TestExtractOperationMatchesEnabledEvent(t *testing.T) {
	t.Parallel()
	status := circuitrun.StatusReport{
		Enabled: []circuitb.CallStatus{
			{Call: "Advance(start)"},
			{Call: "Advance(finish)"},
		},
	}

	if op := ExtractOperation("start", status); op != "start" {
		t.Fatalf("extract start = %q", op)
	}
	if op := ExtractOperation("finish", status); op != "finish" {
		t.Fatalf("extract finish = %q", op)
	}
	if op := ExtractOperation("I think we should start", status); op != "start" {
		t.Fatalf("extract from sentence = %q", op)
	}
}

func TestExtractOperationReturnsEmptyForNoMatch(t *testing.T) {
	t.Parallel()
	status := circuitrun.StatusReport{
		Enabled: []circuitb.CallStatus{{Call: "Advance(start)"}},
	}

	if op := ExtractOperation("bogus", status); op != "" {
		t.Fatalf("extract bogus = %q, want empty", op)
	}
}

func TestExtractOperationIsCaseInsensitive(t *testing.T) {
	t.Parallel()
	status := circuitrun.StatusReport{
		Enabled: []circuitb.CallStatus{{Call: "Advance(requestReview)"}},
	}

	if op := ExtractOperation("REQUESTREVIEW", status); op != "requestReview" {
		t.Fatalf("extract case-insensitive = %q", op)
	}
}

func TestExtractEvent(t *testing.T) {
	t.Parallel()
	if e := ExtractEvent("Advance(start)"); e != "start" {
		t.Fatalf("extract event = %q", e)
	}
	if e := ExtractEvent("noop"); e != "noop" {
		t.Fatalf("extract bare = %q", e)
	}
}

func TestIsTerminal(t *testing.T) {
	t.Parallel()
	terminal := circuitrun.StatusReport{Enabled: nil}
	active := circuitrun.StatusReport{Enabled: []circuitb.CallStatus{{Call: "Advance(start)"}}}

	if !IsTerminal(terminal) {
		t.Fatal("terminal should be true")
	}
	if IsTerminal(active) {
		t.Fatal("active should not be terminal")
	}
}

func TestExtractTextFromMessage(t *testing.T) {
	t.Parallel()
	msg := map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{"type": "text", "text": "hello"},
			map[string]interface{}{"type": "text", "text": "world"},
			map[string]interface{}{"type": "thinking", "thinking": "hmm"},
		},
	}

	text := ExtractTextFromMessage(msg)

	if text != "hello\nworld" {
		t.Fatalf("extract text = %q", text)
	}
}

func TestExtractTextFromMessageHandlesEmpty(t *testing.T) {
	t.Parallel()
	if text := ExtractTextFromMessage(map[string]interface{}{}); text != "" {
		t.Fatalf("empty message text = %q", text)
	}
	if text := ExtractTextFromMessage(map[string]interface{}{"content": "not array"}); text != "" {
		t.Fatalf("non-array content text = %q", text)
	}
}
