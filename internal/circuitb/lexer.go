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
	if current == '<' && lexer.match('-') && lexer.match('-') {
		return lexer.complete(start, tokenReturn, "<--"), nil
	}
	if current == ':' && lexer.match('=') {
		return lexer.complete(start, tokenAssign, ":="), nil
	}
	if current == '/' && lexer.match('=') {
		return lexer.complete(start, tokenNotEquals, "/="), nil
	}
	if current == '<' && lexer.match('=') {
		return lexer.complete(start, tokenLessEqual, "<="), nil
	}
	if current == '>' && lexer.match('=') {
		return lexer.complete(start, tokenGreaterEqual, ">="), nil
	}
	if current == '|' && lexer.match('|') {
		return lexer.complete(start, tokenParallel, "||"), nil
	}
	switch current {
	case ':':
		return lexer.complete(start, tokenColon, ":"), nil
	case ';':
		return lexer.complete(start, tokenSemicolon, ";"), nil
	case ',':
		return lexer.complete(start, tokenComma, ","), nil
	case '(':
		return lexer.complete(start, tokenLParen, "("), nil
	case ')':
		return lexer.complete(start, tokenRParen, ")"), nil
	case '{':
		return lexer.complete(start, tokenLBrace, "{"), nil
	case '}':
		return lexer.complete(start, tokenRBrace, "}"), nil
	case '=':
		return lexer.complete(start, tokenEquals, "="), nil
	case '<':
		return lexer.complete(start, tokenLess, "<"), nil
	case '>':
		return lexer.complete(start, tokenGreater, ">"), nil
	case '&':
		return lexer.complete(start, tokenAmpersand, "&"), nil
	}
	return token{}, Diagnostic{Span: start, Message: fmt.Sprintf("unexpected character %q", current)}
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

func (lexer *lexer) done() bool {
	return lexer.index >= len(lexer.input)
}

func isIdentifierStart(value rune) bool {
	return unicode.IsLetter(value) || value == '_'
}

func isIdentifierPart(value rune) bool {
	return unicode.IsLetter(value) || unicode.IsDigit(value) || value == '_'
}
