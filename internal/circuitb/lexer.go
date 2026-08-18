package circuitb

import (
	"fmt"
	"unicode"
)

type lexer struct {
	file   string
	input  []rune
	index  int
	line   int
	column int
}

func lex(file string, input string) ([]token, error) {
	scanner := lexer{file: file, input: []rune(input), line: 1, column: 1}
	return scanner.tokens()
}

func (lexer *lexer) tokens() ([]token, error) {
	tokens := []token{}
	for !lexer.done() {
		current := lexer.peek()
		if unicode.IsSpace(current) {
			lexer.advance()
			continue
		}
		if isIdentifierStart(current) {
			tokens = append(tokens, lexer.identifier())
			continue
		}
		if unicode.IsDigit(current) {
			tokens = append(tokens, lexer.number())
			continue
		}
		next, err := lexer.operator()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, next)
	}
	tokens = append(tokens, token{typeof: tokenEOF, span: lexer.spanAtCurrent()})
	return tokens, nil
}

func (lexer *lexer) identifier() token {
	start := lexer.spanAtCurrent()
	value := []rune{}
	for !lexer.done() && isIdentifierPart(lexer.peek()) {
		value = append(value, lexer.advance())
	}
	text := string(value)
	typeof, ok := keywordType(text)
	if !ok {
		typeof = tokenIdentifier
	}
	start.EndLine = lexer.line
	start.EndColumn = lexer.column
	return token{typeof: typeof, value: text, span: start}
}

func (lexer *lexer) number() token {
	start := lexer.spanAtCurrent()
	value := []rune{}
	for !lexer.done() && unicode.IsDigit(lexer.peek()) {
		value = append(value, lexer.advance())
	}
	start.EndLine = lexer.line
	start.EndColumn = lexer.column
	return token{typeof: tokenNumber, value: string(value), span: start}
}

func (lexer *lexer) operator() (token, error) {
	start := lexer.spanAtCurrent()
	current := lexer.advance()
	if typeof, value, ok := lexer.compoundOperator(current); ok {
		return lexer.complete(start, typeof, value), nil
	}
	if typeof, value, ok := simpleOperator(current); ok {
		return lexer.complete(start, typeof, value), nil
	}
	return token{}, Diagnostic{Span: start, Message: fmt.Sprintf("unexpected character %q", current)}
}

func (lexer *lexer) tryReturn() bool {
	r0, ok0 := lexer.peekAt(0)
	if !ok0 || r0 != '-' {
		return false
	}
	r1, ok1 := lexer.peekAt(1)
	if !ok1 || r1 != '-' {
		return false
	}
	lexer.advance()
	lexer.advance()
	return true
}

func (lexer *lexer) compoundOperator(current rune) (tokenType, string, bool) {
	// "<--" uses safe two-character lookahead before consuming.
	if current == '<' && lexer.tryReturn() {
		return tokenReturn, "<--", true
	}
	if current == ':' && lexer.match('=') {
		return tokenAssign, ":=", true
	}
	if current == '/' && lexer.match('=') {
		return tokenNotEquals, "/=", true
	}
	if current == '<' && lexer.match('=') {
		return tokenLessEqual, "<=", true
	}
	if current == '>' && lexer.match('=') {
		return tokenGreaterEqual, ">=", true
	}
	if current == '|' && lexer.match('|') {
		return tokenParallel, "||", true
	}
	return tokenEOF, "", false
}

func simpleOperator(current rune) (tokenType, string, bool) {
	switch current {
	case ':':
		return tokenColon, ":", true
	case ';':
		return tokenSemicolon, ";", true
	case ',':
		return tokenComma, ",", true
	case '(':
		return tokenLParen, "(", true
	case ')':
		return tokenRParen, ")", true
	case '{':
		return tokenLBrace, "{", true
	case '}':
		return tokenRBrace, "}", true
	case '=':
		return tokenEquals, "=", true
	case '<':
		return tokenLess, "<", true
	case '>':
		return tokenGreater, ">", true
	case '&':
		return tokenAmpersand, "&", true
	}
	return tokenEOF, "", false
}

func (lexer *lexer) complete(span Span, typeof tokenType, value string) token {
	span.EndLine = lexer.line
	span.EndColumn = lexer.column
	return token{typeof: typeof, value: value, span: span}
}

func (lexer *lexer) spanAtCurrent() Span {
	return Span{File: lexer.file, Line: lexer.line, Column: lexer.column, EndLine: lexer.line, EndColumn: lexer.column}
}

func (lexer *lexer) match(expected rune) bool {
	if lexer.done() || lexer.peek() != expected {
		return false
	}
	lexer.advance()
	return true
}

func (lexer *lexer) advance() rune {
	current := lexer.input[lexer.index]
	lexer.index++
	if current == '\n' {
		lexer.line++
		lexer.column = 1
	} else {
		lexer.column++
	}
	return current
}

func (lexer *lexer) peek() rune {
	return lexer.input[lexer.index]
}

func (lexer *lexer) peekAt(offset int) (rune, bool) {
	index := lexer.index + offset
	if index >= len(lexer.input) {
		return 0, false
	}
	return lexer.input[index], true
}

func (lexer *lexer) done() bool {
	return lexer.index >= len(lexer.input)
}

func isIdentifierStart(value rune) bool {
	return unicode.IsLetter(value) || value == '_'
}

func isIdentifierPart(value rune) bool {
	return unicode.IsLetter(value) || unicode.IsDigit(value) || value == '_'
}
