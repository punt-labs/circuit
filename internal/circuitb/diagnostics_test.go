package circuitb

import (
	"strings"
	"testing"
)

func TestSpanStringWithFile(t *testing.T) {
	t.Parallel()
	s := Span{File: "test.mch", Line: 3, Column: 5}
	if s.String() != "test.mch:3:5" {
		t.Fatalf("span = %q", s.String())
	}
}

func TestSpanStringWithoutFile(t *testing.T) {
	t.Parallel()
	s := Span{Line: 7, Column: 2}
	if s.String() != "7:2" {
		t.Fatalf("span = %q", s.String())
	}
}

func TestDiagnosticErrorWithoutHint(t *testing.T) {
	t.Parallel()
	d := Diagnostic{Span: Span{File: "test.mch", Line: 3, Column: 5}, Message: "bad token"}
	e := d.Error()
	if !strings.Contains(e, "test.mch:3:5") || !strings.Contains(e, "bad token") {
		t.Fatalf("diagnostic = %q", e)
	}
	if strings.Contains(e, "hint") {
		t.Fatalf("diagnostic has unexpected hint: %q", e)
	}
}

func TestDiagnosticErrorWithHint(t *testing.T) {
	t.Parallel()
	d := Diagnostic{Span: Span{File: "test.mch", Line: 1, Column: 1}, Message: "unsupported", Hint: "use IF instead"}
	e := d.Error()
	if !strings.Contains(e, "unsupported") || !strings.Contains(e, "use IF instead") {
		t.Fatalf("diagnostic = %q", e)
	}
}

func TestDiagnosticsErrNilWhenEmpty(t *testing.T) {
	t.Parallel()
	var ds Diagnostics
	if ds.Err() != nil {
		t.Fatal("empty diagnostics Err != nil")
	}
}

func TestDiagnosticsErrNonNilWhenPopulated(t *testing.T) {
	t.Parallel()
	ds := Diagnostics{{Span: Span{File: "x.mch", Line: 1, Column: 1}, Message: "a"}}
	err := ds.Err()
	if err == nil {
		t.Fatal("diagnostics Err = nil")
	}
	if !strings.Contains(err.Error(), "a") {
		t.Fatalf("diagnostics error = %q", err.Error())
	}
}

func TestDiagnosticsErrorConcatenatesMultiple(t *testing.T) {
	t.Parallel()
	ds := Diagnostics{
		{Span: Span{Line: 1, Column: 1}, Message: "first"},
		{Span: Span{Line: 2, Column: 1}, Message: "second"},
	}
	e := ds.Error()
	if !strings.Contains(e, "first") || !strings.Contains(e, "second") {
		t.Fatalf("diagnostics error = %q", e)
	}
}

func TestDiagnosticsErrorEmptyString(t *testing.T) {
	t.Parallel()
	var ds Diagnostics
	if ds.Error() != "" {
		t.Fatalf("empty diagnostics Error = %q", ds.Error())
	}
}
