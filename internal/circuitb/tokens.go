package circuitb

type tokenType int

const (
	tokenEOF tokenType = iota
	tokenIdentifier
	tokenNumber
	tokenMachine
	tokenSets
	tokenVariables
	tokenInvariant
	tokenInitialisation
	tokenOperations
	tokenPre
	tokenThen
	tokenIf
	tokenElsif
	tokenElse
	tokenEnd
	tokenOr
	tokenNot
	tokenAny
	tokenColon
	tokenSemicolon
	tokenComma
	tokenLParen
	tokenRParen
	tokenLBrace
	tokenRBrace
	tokenEquals
	tokenNotEquals
	tokenLess
	tokenLessEqual
	tokenGreater
	tokenGreaterEqual
	tokenAssign
	tokenParallel
	tokenAmpersand
	tokenReturn
)

type token struct {
	typeof tokenType
	value  string
	span   Span
}

// uppercaseKeywords maps uppercase B-method keywords to their token types.
var uppercaseKeywords = map[string]tokenType{
	"MACHINE":        tokenMachine,
	"SETS":           tokenSets,
	"VARIABLES":      tokenVariables,
	"INVARIANT":      tokenInvariant,
	"INITIALISATION": tokenInitialisation,
	"OPERATIONS":     tokenOperations,
	"PRE":            tokenPre,
	"THEN":           tokenThen,
	"IF":             tokenIf,
	"ELSIF":          tokenElsif,
	"ELSE":           tokenElse,
	"END":            tokenEnd,
	"ANY":            tokenAny,
}

// lowercaseKeywords maps lowercase B-method keywords to their token types.
var lowercaseKeywords = map[string]tokenType{
	"or":  tokenOr,
	"not": tokenNot,
}

func keywordType(value string) (tokenType, bool) {
	if tt, ok := uppercaseKeywords[value]; ok {
		return tt, true
	}
	if tt, ok := lowercaseKeywords[value]; ok {
		return tt, true
	}
	return tokenEOF, false
}
