package circuitrpc

import (
	"fmt"
	"strings"

	"github.com/punt-labs/circuit/internal/circuitrun"
)

func FormatPrompt(status circuitrun.StatusReport) string {
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

func ExtractOperation(response string, status circuitrun.StatusReport) string {
	lower := strings.ToLower(strings.TrimSpace(response))
	for _, call := range status.Enabled {
		event := ExtractEvent(call.Call)
		if strings.Contains(lower, strings.ToLower(event)) {
			return event
		}
	}
	return ""
}

func ExtractEvent(call string) string {
	start := strings.Index(call, "(")
	end := strings.Index(call, ")")
	if start >= 0 && end > start {
		return call[start+1 : end]
	}
	return call
}

func IsTerminal(status circuitrun.StatusReport) bool {
	return len(status.Enabled) == 0
}

func ExtractTextFromMessage(msg map[string]interface{}) string {
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
