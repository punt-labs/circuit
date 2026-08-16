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

func keywordType(value string) (tokenType, bool) {
	switch value {
	case "MACHINE":
		return tokenMachine, true
	case "SETS":
		return tokenSets, true
	case "VARIABLES":
		return tokenVariables, true
	case "INVARIANT":
		return tokenInvariant, true
	case "INITIALISATION":
		return tokenInitialisation, true
	case "OPERATIONS":
		return tokenOperations, true
	case "PRE":
		return tokenPre, true
	case "THEN":
		return tokenThen, true
	case "IF":
		return tokenIf, true
	case "ELSIF":
		return tokenElsif, true
	case "ELSE":
		return tokenElse, true
	case "END":
		return tokenEnd, true
	case "or":
		return tokenOr, true
	case "not":
		return tokenNot, true
	case "ANY":
		return tokenAny, true
	}
	return tokenEOF, false
}
