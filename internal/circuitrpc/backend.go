package circuitrpc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

// PiRPCBackend adapts a pi --mode rpc process (or any duplex JSON stream) to
// AgentBackend. Each Prompt writes a prompt request and reads events until an
// agent_settled event, returning the last assistant message text.
type PiRPCBackend struct {
	writer io.Writer
	reader *bufio.Reader
	seq    int
}

func NewPiRPCBackend(writer io.Writer, reader *bufio.Reader) *PiRPCBackend {
	return &PiRPCBackend{writer: writer, reader: reader}
}

func (backend *PiRPCBackend) Prompt(message string) (string, error) {
	backend.seq++
	request := PromptRequest{ID: fmt.Sprintf("req-%d", backend.seq), Type: "prompt", Message: message}
	data, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	if _, err := fmt.Fprintf(backend.writer, "%s\n", data); err != nil {
		return "", err
	}

	var lastText string
	for {
		line, err := backend.reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.Type == "message_end" && event.Message.Role == "assistant" {
			lastText = ExtractTextFromMessage(event.Message)
		}
		if event.Type == "agent_settled" {
			return lastText, nil
		}
	}
}
