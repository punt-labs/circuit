package circuitb

import (
	"strings"
	"testing"
)

func TestLexIdentifiers(t *testing.T) {
	t.Parallel()
	tokens, err := lex("test", "MACHINE Foo current")
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	expect := []struct {
		typeof tokenType
		value  string
	}{
		{tokenMachine, "MACHINE"},
		{tokenIdentifier, "Foo"},
		{tokenIdentifier, "current"},
		{tokenEOF, ""},
	}
	if len(tokens) != len(expect) {
		t.Fatalf("tokens = %d, want %d", len(tokens), len(expect))
	}
	for i, e := range expect {
		if tokens[i].typeof != e.typeof || tokens[i].value != e.value {
			t.Errorf("token[%d] = %v/%q, want %v/%q", i, tokens[i].typeof, tokens[i].value, e.typeof, e.value)
		}
	}
}

func TestLexKeywords(t *testing.T) {
	t.Parallel()
	keywords := []string{
		"MACHINE", "SETS", "VARIABLES", "INVARIANT",
		"INITIALISATION", "OPERATIONS", "PRE", "THEN",
		"IF", "ELSIF", "ELSE", "END", "ANY",
	}
	for _, kw := range keywords {
		tokens, err := lex("test", kw)
		if err != nil {
			t.Fatalf("lex %s: %v", kw, err)
		}
		if tokens[0].value != kw {
			t.Errorf("keyword %s lexed as %q", kw, tokens[0].value)
		}
		kt, ok := keywordType(kw)
		if !ok {
			t.Errorf("keywordType(%s) = false", kw)
		}
		if tokens[0].typeof != kt {
			t.Errorf("keyword %s type = %v, want %v", kw, tokens[0].typeof, kt)
		}
	}
}

func TestLexNumbers(t *testing.T) {
	t.Parallel()
	tokens, err := lex("test", "0 42 100")
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	values := []string{"0", "42", "100"}
	for i, v := range values {
		if tokens[i].typeof != tokenNumber || tokens[i].value != v {
			t.Errorf("token[%d] = %v/%q, want number/%q", i, tokens[i].typeof, tokens[i].value, v)
		}
	}
}

func TestLexOperators(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input  string
		typeof tokenType
		value  string
	}{
		{"=", tokenEquals, "="},
		{"/=", tokenNotEquals, "/="},
		{":=", tokenAssign, ":="},
		{":", tokenColon, ":"},
		{";", tokenSemicolon, ";"},
		{",", tokenComma, ","},
		{"&", tokenAmpersand, "&"},
		{"(", tokenLParen, "("},
		{")", tokenRParen, ")"},
		{"{", tokenLBrace, "{"},
		{"}", tokenRBrace, "}"},
		{"||", tokenParallel, "||"},
		{"<", tokenLess, "<"},
		{">", tokenGreater, ">"},
		{"<=", tokenLessEqual, "<="},
		{">=", tokenGreaterEqual, ">="},
	}
	for _, c := range cases {
		tokens, err := lex("test", c.input)
		if err != nil {
			t.Fatalf("lex %q: %v", c.input, err)
		}
		if tokens[0].typeof != c.typeof || tokens[0].value != c.value {
			t.Errorf("lex %q = %v/%q, want %v/%q", c.input, tokens[0].typeof, tokens[0].value, c.typeof, c.value)
		}
	}
}

func TestLexSourceSpans(t *testing.T) {
	t.Parallel()
	tokens, err := lex("test.mch", "MACHINE Foo")
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	if tokens[0].span.File != "test.mch" {
		t.Errorf("span file = %q", tokens[0].span.File)
	}
	if tokens[0].span.Line != 1 || tokens[0].span.Column != 1 {
		t.Errorf("MACHINE span = %d:%d, want 1:1", tokens[0].span.Line, tokens[0].span.Column)
	}
	if tokens[1].span.Column != 9 {
		t.Errorf("Foo column = %d, want 9", tokens[1].span.Column)
	}
}

func TestLexNewlineTracking(t *testing.T) {
	t.Parallel()
	tokens, err := lex("test", "A\nB\nC")
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	lines := []int{1, 2, 3}
	for i, line := range lines {
		if tokens[i].span.Line != line {
			t.Errorf("token[%d] line = %d, want %d", i, tokens[i].span.Line, line)
		}
	}
}

func TestLexWhitespaceIgnored(t *testing.T) {
	t.Parallel()
	tokens, err := lex("test", "  A   B  \t C  ")
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	if len(tokens) != 4 { // A B C EOF
		t.Fatalf("tokens = %d, want 4", len(tokens))
	}
}

func TestLexEmptyInput(t *testing.T) {
	t.Parallel()
	tokens, err := lex("test", "")
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	if len(tokens) != 1 || tokens[0].typeof != tokenEOF {
		t.Fatalf("empty input tokens = %v", tokens)
	}
}

func TestLexInvalidCharacter(t *testing.T) {
	t.Parallel()
	_, err := lex("test", "A @ B")
	if err == nil {
		t.Fatal("invalid character returned nil error")
	}
	if !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("error = %v, want unexpected", err)
	}
}
