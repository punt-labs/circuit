package circuitb

import "strings"

import "fmt"

type Span struct {
	File      string
	Line      int
	Column    int
	EndLine   int
	EndColumn int
}

func (span Span) String() string {
	if span.File == "" {
		return fmt.Sprintf("%d:%d", span.Line, span.Column)
	}
	return fmt.Sprintf("%s:%d:%d", span.File, span.Line, span.Column)
}

type Diagnostic struct {
	Span    Span
	Message string
	Hint    string
}

func (diagnostic Diagnostic) Error() string {
	if diagnostic.Hint == "" {
		return diagnostic.Span.String() + ": " + diagnostic.Message
	}
	return diagnostic.Span.String() + ": " + diagnostic.Message + "\n  hint: " + diagnostic.Hint
}

type Diagnostics []Diagnostic

func (diagnostics Diagnostics) Error() string {
	if len(diagnostics) == 0 {
		return ""
	}
	var message strings.Builder
	message.WriteString(diagnostics[0].Error())
	for _, diagnostic := range diagnostics[1:] {
		message.WriteString("\n" + diagnostic.Error())
	}
	return message.String()
}

func (diagnostics Diagnostics) Err() error {
	if len(diagnostics) == 0 {
		return nil
	}
	return diagnostics
}
