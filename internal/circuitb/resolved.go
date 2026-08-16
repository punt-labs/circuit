package circuitb

type valueKind int

const (
	valueEnum valueKind = iota
	valueNat
	valueBool
)

type value struct {
	kind valueKind
	enum string
	nat  int
	bool bool
}

type variable struct {
	name string
	kind valueKind
	set  string
}

type Machine struct {
	Name          string
	sets          map[string][]string
	variables     map[string]variable
	initial       map[string]value
	operations    []operation
	advance       operation
	currentName   string
	stateSetName  string
	transitionSet string
}

type operation struct {
	Name       string
	Parameters []string
	Pre        predicate
	Body       substitution
	Span       Span
}

type predicate interface {
	evaluate(values map[string]value, bindings map[string]value) bool
	explain(values map[string]value, bindings map[string]value) []string
}

type binaryPredicate struct {
	operator string
	left     predicate
	right    predicate
}

type comparisonPredicate struct {
	operator string
	left     expression
	right    expression
}

type membershipPredicate struct {
	element expression
	set     setExpression
}

type notPredicate struct {
	inner predicate
}

type expression interface {
	evaluate(values map[string]value, bindings map[string]value) value
	format() string
}

type identifierExpression struct {
	name string
}

type numberExpression struct {
	value int
}

type setExpression interface {
	contains(candidate value, values map[string]value, bindings map[string]value) bool
	format() string
}

type namedSetExpression struct {
	name   string
	values []string
}

type literalSetExpression struct {
	values []expression
}

type substitution interface {
	apply(values map[string]value, bindings map[string]value) map[string]value
}

type assignment struct {
	name  string
	value expression
}

type parallelAssignment struct {
	assignments []assignment
}

type ifSubstitution struct {
	branches []guardedSubstitution
	fallback substitution
}

type guardedSubstitution struct {
	condition predicate
	body      substitution
}

type StateReport struct {
	Current string
	Enabled []CallStatus
	Blocked []CallStatus
}

type CallStatus struct {
	Call   string
	Failed []string
}

type AdvanceResult struct {
	Allowed bool
	From    string
	To      string
	Failed  []string
}
