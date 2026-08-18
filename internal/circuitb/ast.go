package circuitb

type rawMachine struct {
	Name              string
	Sets              []rawSet
	Variables         []rawIdentifier
	HasInvariant      bool
	Invariant         rawPredicate
	HasInitialisation bool
	Initialisation    rawSubstitution
	HasOperations     bool
	Operations        []rawOperation
	Span              Span
}

type rawSet struct {
	Name   string
	Values []rawIdentifier
	Span   Span
}

type rawIdentifier struct {
	Name string
	Span Span
}

type rawOperation struct {
	Name       string
	Parameters []rawIdentifier
	Output     *rawIdentifier
	Pre        rawPredicate
	Body       rawSubstitution
	Span       Span
}

type rawPredicate interface {
	predicateSpan() Span
}

type rawBinaryPredicate struct {
	Operator string
	Left     rawPredicate
	Right    rawPredicate
	Span     Span
}

func (predicate rawBinaryPredicate) predicateSpan() Span {
	return predicate.Span
}

type rawComparisonPredicate struct {
	Operator string
	Left     rawExpression
	Right    rawExpression
	Span     Span
}

func (predicate rawComparisonPredicate) predicateSpan() Span {
	return predicate.Span
}

type rawNotPredicate struct {
	Inner rawPredicate
	Span  Span
}

func (predicate rawNotPredicate) predicateSpan() Span {
	return predicate.Span
}

type rawMembershipPredicate struct {
	Element rawExpression
	Set     rawSetExpression
	Span    Span
}

func (predicate rawMembershipPredicate) predicateSpan() Span {
	return predicate.Span
}

type rawExpression interface {
	expressionSpan() Span
}

type rawIdentifierExpression struct {
	Name string
	Span Span
}

func (expression rawIdentifierExpression) expressionSpan() Span {
	return expression.Span
}

type rawNumberExpression struct {
	Value int
	Span  Span
}

func (expression rawNumberExpression) expressionSpan() Span {
	return expression.Span
}

type rawSetExpression interface {
	setExpressionSpan() Span
}

type rawNamedSetExpression struct {
	Name string
	Span Span
}

func (expression rawNamedSetExpression) setExpressionSpan() Span {
	return expression.Span
}

type rawLiteralSetExpression struct {
	Values []rawExpression
	Span   Span
}

func (expression rawLiteralSetExpression) setExpressionSpan() Span {
	return expression.Span
}

type rawSubstitution interface {
	substitutionSpan() Span
}

type rawUnsupportedSubstitution struct {
	Construct string
	Span      Span
}

func (substitution rawUnsupportedSubstitution) substitutionSpan() Span {
	return substitution.Span
}

type rawAssignment struct {
	Name  string
	Value rawExpression
	Span  Span
}

func (substitution rawAssignment) substitutionSpan() Span {
	return substitution.Span
}

type rawParallelAssignment struct {
	Assignments []rawAssignment
	Span        Span
}

func (substitution rawParallelAssignment) substitutionSpan() Span {
	return substitution.Span
}

type rawIfSubstitution struct {
	Branches []rawGuardedSubstitution
	Else     rawSubstitution
	Span     Span
}

func (substitution rawIfSubstitution) substitutionSpan() Span {
	return substitution.Span
}

type rawGuardedSubstitution struct {
	Condition rawPredicate
	Body      rawSubstitution
}
