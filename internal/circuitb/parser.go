package circuitb

import (
	"fmt"
	"strconv"
)

type parser struct {
	tokens []token
	index  int
}

func parse(tokens []token) (rawMachine, error) {
	parser := parser{tokens: tokens}
	machine, err := parser.machine()
	if err != nil {
		return rawMachine{}, err
	}
	if _, err := parser.expect(tokenEOF, "expected end of file"); err != nil {
		return rawMachine{}, err
	}
	return machine, nil
}

func (parser *parser) machine() (rawMachine, error) {
	start, err := parser.expect(tokenMachine, "expected MACHINE")
	if err != nil {
		return rawMachine{}, err
	}
	name, err := parser.expect(tokenIdentifier, "expected machine name")
	if err != nil {
		return rawMachine{}, err
	}
	machine := rawMachine{Name: name.value, Span: start.span}
	for !parser.at(tokenEnd) && !parser.at(tokenEOF) {
		switch {
		case parser.match(tokenSets):
			sets, err := parser.sets()
			if err != nil {
				return rawMachine{}, err
			}
			machine.Sets = sets
		case parser.match(tokenVariables):
			variables, err := parser.variableList()
			if err != nil {
				return rawMachine{}, err
			}
			machine.Variables = variables
		case parser.match(tokenInvariant):
			invariant, err := parser.predicateUntilClause()
			if err != nil {
				return rawMachine{}, err
			}
			machine.HasInvariant = true
			machine.Invariant = invariant
		case parser.match(tokenInitialisation):
			initialisation, err := parser.substitutionUntilClause()
			if err != nil {
				return rawMachine{}, err
			}
			machine.HasInitialisation = true
			machine.Initialisation = initialisation
		case parser.match(tokenOperations):
			operations, err := parser.operations()
			if err != nil {
				return rawMachine{}, err
			}
			machine.HasOperations = true
			machine.Operations = operations
		default:
			return rawMachine{}, Diagnostic{Span: parser.current().span, Message: fmt.Sprintf("unexpected %s in machine body", parser.current().value)}
		}
	}
	end, err := parser.expect(tokenEnd, "expected END")
	if err != nil {
		return rawMachine{}, err
	}
	machine.Span.EndLine = end.span.EndLine
	machine.Span.EndColumn = end.span.EndColumn
	return machine, nil
}

func isClauseStart(typeof tokenType) bool {
	return typeof == tokenSets || typeof == tokenVariables ||
		typeof == tokenInvariant || typeof == tokenInitialisation ||
		typeof == tokenOperations || typeof == tokenEnd
}

func (parser *parser) variableList() ([]rawIdentifier, error) {
	identifiers := []rawIdentifier{}
	for !isClauseStart(parser.current().typeof) && !parser.at(tokenEOF) {
		identifier, err := parser.expect(tokenIdentifier, "expected variable name")
		if err != nil {
			return nil, err
		}
		identifiers = append(identifiers, rawIdentifier{Name: identifier.value, Span: identifier.span})
		parser.consume(tokenComma)
	}
	return identifiers, nil
}

func (parser *parser) predicateUntilClause() (rawPredicate, error) {
	predicate, err := parser.disjunction()
	if err != nil {
		return nil, err
	}
	for !isClauseStart(parser.current().typeof) && !parser.at(tokenEOF) {
		if !parser.consume(tokenAmpersand) {
			break
		}
		right, err := parser.disjunction()
		if err != nil {
			return nil, err
		}
		span := predicate.predicateSpan()
		span.EndLine = right.predicateSpan().EndLine
		span.EndColumn = right.predicateSpan().EndColumn
		predicate = rawBinaryPredicate{Operator: "&", Left: predicate, Right: right, Span: span}
	}
	return predicate, nil
}

func (parser *parser) substitutionUntilClause() (rawSubstitution, error) {
	if parser.consume(tokenIf) {
		return parser.ifSubstitution()
	}
	if parser.consume(tokenAny) {
		return parser.unsupportedSubstitution("ANY")
	}
	first, err := parser.assignment()
	if err != nil {
		return nil, err
	}
	assignments := []rawAssignment{first}
	for parser.consume(tokenParallel) {
		next, err := parser.assignment()
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, next)
	}
	if len(assignments) == 1 {
		return assignments[0], nil
	}
	span := assignments[0].Span
	last := assignments[len(assignments)-1].Span
	span.EndLine = last.EndLine
	span.EndColumn = last.EndColumn
	return rawParallelAssignment{Assignments: assignments, Span: span}, nil
}

func (parser *parser) sets() ([]rawSet, error) {
	sets := []rawSet{}
	for !isClauseStart(parser.current().typeof) && !parser.at(tokenEOF) {
		name, err := parser.expect(tokenIdentifier, "expected set name")
		if err != nil {
			return nil, err
		}
		if _, err := parser.expect(tokenEquals, "expected = after set name"); err != nil {
			return nil, err
		}
		if _, err := parser.expect(tokenLBrace, "expected { in enumerated set"); err != nil {
			return nil, err
		}
		values, err := parser.identifierList(tokenRBrace, "expected set value")
		if err != nil {
			return nil, err
		}
		end, err := parser.expect(tokenRBrace, "expected } after set values")
		if err != nil {
			return nil, err
		}
		span := name.span
		span.EndLine = end.span.EndLine
		span.EndColumn = end.span.EndColumn
		sets = append(sets, rawSet{Name: name.value, Values: values, Span: span})
		parser.consume(tokenSemicolon)
	}
	return sets, nil
}

func (parser *parser) operations() ([]rawOperation, error) {
	operations := []rawOperation{}
	for !parser.at(tokenEnd) && !parser.at(tokenEOF) {
		operation, err := parser.operation()
		if err != nil {
			return nil, err
		}
		operations = append(operations, operation)
		parser.consume(tokenSemicolon)
	}
	return operations, nil
}

func (parser *parser) operation() (rawOperation, error) {
	first, err := parser.expectAny([]tokenType{tokenIdentifier}, "expected operation name or output variable")
	if err != nil {
		return rawOperation{}, err
	}
	operation := rawOperation{Name: first.value, Span: first.span}
	if parser.consume(tokenReturn) {
		operation.Output = &rawIdentifier{Name: first.value, Span: first.span}
		name, err := parser.expect(tokenIdentifier, "expected query operation name")
		if err != nil {
			return rawOperation{}, err
		}
		operation.Name = name.value
	}
	if parser.consume(tokenLParen) {
		parameters, err := parser.identifierList(tokenRParen, "expected parameter name")
		if err != nil {
			return rawOperation{}, err
		}
		operation.Parameters = parameters
		if _, err := parser.expect(tokenRParen, "expected ) after parameters"); err != nil {
			return rawOperation{}, err
		}
	}
	if _, err := parser.expect(tokenEquals, "expected = in operation"); err != nil {
		return rawOperation{}, err
	}
	if parser.consume(tokenPre) {
		pre, err := parser.predicateUntil(tokenThen)
		if err != nil {
			return rawOperation{}, err
		}
		operation.Pre = pre
		if _, err := parser.expect(tokenThen, "expected THEN in operation"); err != nil {
			return rawOperation{}, err
		}
		body, err := parser.substitutionUntil(tokenEnd)
		if err != nil {
			return rawOperation{}, err
		}
		operation.Body = body
	} else {
		body, err := parser.substitutionUntil(tokenEnd)
		if err != nil {
			return rawOperation{}, err
		}
		operation.Body = body
	}
	end, err := parser.expect(tokenEnd, "expected END after operation")
	if err != nil {
		return rawOperation{}, err
	}
	operation.Span.EndLine = end.span.EndLine
	operation.Span.EndColumn = end.span.EndColumn
	return operation, nil
}

func (parser *parser) predicateUntil(end tokenType) (rawPredicate, error) {
	predicate, err := parser.disjunction()
	if err != nil {
		return nil, err
	}
	if !parser.at(end) {
		return nil, Diagnostic{Span: parser.current().span, Message: fmt.Sprintf("expected %s after predicate", tokenName(end))}
	}
	return predicate, nil
}

func (parser *parser) disjunction() (rawPredicate, error) {
	left, err := parser.conjunction()
	if err != nil {
		return nil, err
	}
	for parser.consume(tokenOr) {
		operator := parser.previous()
		right, err := parser.conjunction()
		if err != nil {
			return nil, err
		}
		span := left.predicateSpan()
		span.EndLine = right.predicateSpan().EndLine
		span.EndColumn = right.predicateSpan().EndColumn
		left = rawBinaryPredicate{Operator: operator.value, Left: left, Right: right, Span: span}
	}
	return left, nil
}

func (parser *parser) conjunction() (rawPredicate, error) {
	left, err := parser.atom()
	if err != nil {
		return nil, err
	}
	for parser.consume(tokenAmpersand) {
		operator := parser.previous()
		right, err := parser.atom()
		if err != nil {
			return nil, err
		}
		span := left.predicateSpan()
		span.EndLine = right.predicateSpan().EndLine
		span.EndColumn = right.predicateSpan().EndColumn
		left = rawBinaryPredicate{Operator: operator.value, Left: left, Right: right, Span: span}
	}
	return left, nil
}

func (parser *parser) atom() (rawPredicate, error) {
	if parser.consume(tokenLParen) {
		predicate, err := parser.disjunction()
		if err != nil {
			return nil, err
		}
		if _, err := parser.expect(tokenRParen, "expected ) after predicate"); err != nil {
			return nil, err
		}
		return predicate, nil
	}
	left, err := parser.expression()
	if err != nil {
		return nil, err
	}
	if parser.consume(tokenColon) {
		setExpression, err := parser.setExpression()
		if err != nil {
			return nil, err
		}
		span := left.expressionSpan()
		span.EndLine = setExpression.setExpressionSpan().EndLine
		span.EndColumn = setExpression.setExpressionSpan().EndColumn
		return rawMembershipPredicate{Element: left, Set: setExpression, Span: span}, nil
	}
	operator, err := parser.expectAny([]tokenType{tokenEquals, tokenNotEquals, tokenLess, tokenLessEqual, tokenGreater, tokenGreaterEqual}, "expected predicate operator")
	if err != nil {
		return nil, err
	}
	right, err := parser.expression()
	if err != nil {
		return nil, err
	}
	span := left.expressionSpan()
	span.EndLine = right.expressionSpan().EndLine
	span.EndColumn = right.expressionSpan().EndColumn
	return rawComparisonPredicate{Operator: operator.value, Left: left, Right: right, Span: span}, nil
}

func (parser *parser) setExpression() (rawSetExpression, error) {
	if parser.consume(tokenIdentifier) {
		identifier := parser.previous()
		return rawNamedSetExpression{Name: identifier.value, Span: identifier.span}, nil
	}
	start, err := parser.expect(tokenLBrace, "expected set name or set literal")
	if err != nil {
		return nil, err
	}
	values := []rawExpression{}
	for !parser.at(tokenRBrace) && !parser.at(tokenEOF) {
		value, err := parser.expression()
		if err != nil {
			return nil, err
		}
		values = append(values, value)
		if !parser.consume(tokenComma) {
			break
		}
	}
	end, err := parser.expect(tokenRBrace, "expected } after set literal")
	if err != nil {
		return nil, err
	}
	span := start.span
	span.EndLine = end.span.EndLine
	span.EndColumn = end.span.EndColumn
	return rawLiteralSetExpression{Values: values, Span: span}, nil
}

func (parser *parser) expression() (rawExpression, error) {
	if parser.consume(tokenIdentifier) {
		identifier := parser.previous()
		return rawIdentifierExpression{Name: identifier.value, Span: identifier.span}, nil
	}
	if parser.consume(tokenNumber) {
		number := parser.previous()
		value, err := strconv.Atoi(number.value)
		if err != nil {
			return nil, Diagnostic{Span: number.span, Message: "invalid integer literal"}
		}
		return rawNumberExpression{Value: value, Span: number.span}, nil
	}
	return nil, Diagnostic{Span: parser.current().span, Message: "expected expression"}
}

func (parser *parser) substitutionUntil(end tokenType) (rawSubstitution, error) {
	return parser.substitutionUntilAny([]tokenType{end})
}

func (parser *parser) ifSubstitution() (rawSubstitution, error) {
	start := parser.previous()
	condition, err := parser.predicateUntil(tokenThen)
	if err != nil {
		return nil, err
	}
	if _, err := parser.expect(tokenThen, "expected THEN in IF"); err != nil {
		return nil, err
	}
	body, err := parser.substitutionUntilAny([]tokenType{tokenElsif, tokenElse, tokenEnd})
	if err != nil {
		return nil, err
	}
	branches := []rawGuardedSubstitution{{Condition: condition, Body: body}}
	for parser.consume(tokenElsif) {
		elsifCondition, err := parser.predicateUntil(tokenThen)
		if err != nil {
			return nil, err
		}
		if _, err := parser.expect(tokenThen, "expected THEN in ELSIF"); err != nil {
			return nil, err
		}
		elsifBody, err := parser.substitutionUntilAny([]tokenType{tokenElsif, tokenElse, tokenEnd})
		if err != nil {
			return nil, err
		}
		branches = append(branches, rawGuardedSubstitution{Condition: elsifCondition, Body: elsifBody})
	}
	var otherwise rawSubstitution
	if parser.consume(tokenElse) {
		otherwise, err = parser.substitutionUntil(tokenEnd)
		if err != nil {
			return nil, err
		}
	}
	end, err := parser.expect(tokenEnd, "expected END after IF")
	if err != nil {
		return nil, err
	}
	span := start.span
	span.EndLine = end.span.EndLine
	span.EndColumn = end.span.EndColumn
	return rawIfSubstitution{Branches: branches, Else: otherwise, Span: span}, nil
}

func (parser *parser) substitutionUntilAny(ends []tokenType) (rawSubstitution, error) {
	if parser.consume(tokenIf) {
		return parser.ifSubstitution()
	}
	if parser.consume(tokenAny) {
		return parser.unsupportedSubstitution("ANY")
	}
	first, err := parser.assignment()
	if err != nil {
		return nil, err
	}
	assignments := []rawAssignment{first}
	for parser.consume(tokenParallel) {
		next, err := parser.assignment()
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, next)
	}
	for _, end := range ends {
		if parser.at(end) {
			if len(assignments) == 1 {
				return assignments[0], nil
			}
			span := assignments[0].Span
			last := assignments[len(assignments)-1].Span
			span.EndLine = last.EndLine
			span.EndColumn = last.EndColumn
			return rawParallelAssignment{Assignments: assignments, Span: span}, nil
		}
	}
	return nil, Diagnostic{Span: parser.current().span, Message: "expected end of substitution"}
}

func (parser *parser) unsupportedSubstitution(construct string) (rawSubstitution, error) {
	start := parser.previous().span
	depth := 0
	for !parser.at(tokenEOF) {
		if parser.consume(tokenEnd) {
			if depth == 0 {
				start.EndLine = parser.previous().span.EndLine
				start.EndColumn = parser.previous().span.EndColumn
				return rawUnsupportedSubstitution{Construct: construct, Span: start}, nil
			}
			depth--
			continue
		}
		if parser.at(tokenIf) || parser.at(tokenAny) {
			depth++
		}
		parser.advance()
	}
	return nil, Diagnostic{Span: start, Message: "unterminated unsupported substitution"}
}

func (parser *parser) assignment() (rawAssignment, error) {
	name, err := parser.expect(tokenIdentifier, "expected assignment target")
	if err != nil {
		return rawAssignment{}, err
	}
	if _, err := parser.expect(tokenAssign, "expected := in assignment"); err != nil {
		return rawAssignment{}, err
	}
	value, err := parser.expression()
	if err != nil {
		return rawAssignment{}, err
	}
	span := name.span
	span.EndLine = value.expressionSpan().EndLine
	span.EndColumn = value.expressionSpan().EndColumn
	return rawAssignment{Name: name.value, Value: value, Span: span}, nil
}

func (parser *parser) identifierList(end tokenType, message string) ([]rawIdentifier, error) {
	identifiers := []rawIdentifier{}
	for !parser.at(end) && !parser.at(tokenEOF) {
		identifier, err := parser.expect(tokenIdentifier, message)
		if err != nil {
			return nil, err
		}
		identifiers = append(identifiers, rawIdentifier{Name: identifier.value, Span: identifier.span})
		if !parser.consume(tokenComma) {
			break
		}
	}
	return identifiers, nil
}

func (parser *parser) expectAny(types []tokenType, message string) (token, error) {
	for _, typeof := range types {
		if parser.at(typeof) {
			return parser.advance(), nil
		}
	}
	return token{}, Diagnostic{Span: parser.current().span, Message: message}
}

func (parser *parser) expect(typeof tokenType, message string) (token, error) {
	if parser.at(typeof) {
		return parser.advance(), nil
	}
	return token{}, Diagnostic{Span: parser.current().span, Message: message}
}

func (parser *parser) consume(typeof tokenType) bool {
	if !parser.at(typeof) {
		return false
	}
	parser.advance()
	return true
}

func (parser *parser) match(typeof tokenType) bool {
	return parser.consume(typeof)
}

func (parser *parser) at(typeof tokenType) bool {
	return parser.current().typeof == typeof
}

func (parser *parser) current() token {
	return parser.tokens[parser.index]
}

func (parser *parser) previous() token {
	return parser.tokens[parser.index-1]
}

func (parser *parser) advance() token {
	current := parser.current()
	parser.index++
	return current
}

func tokenName(typeof tokenType) string {
	switch typeof {
	case tokenThen:
		return "THEN"
	case tokenEnd:
		return "END"
	case tokenInitialisation:
		return "INITIALISATION"
	case tokenOperations:
		return "OPERATIONS"
	case tokenInvariant:
		return "INVARIANT"
	}
	return fmt.Sprintf("token %d", typeof)
}
