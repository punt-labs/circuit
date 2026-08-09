package circuitb

import "fmt"

func resolve(raw rawMachine) (Machine, error) {
	resolver := resolver{
		raw:       raw,
		sets:      map[string][]string{},
		variables: map[string]variable{},
		enums:     map[string]string{},
	}
	return resolver.resolve()
}

type resolver struct {
	raw         rawMachine
	sets        map[string][]string
	variables   map[string]variable
	enums       map[string]string
	diagnostics Diagnostics
}

func (resolver *resolver) resolve() (Machine, error) {
	resolver.resolveSets()
	resolver.resolveVariables()
	initial := resolver.resolveInitialisation()
	operations := resolver.resolveOperations()
	machine := Machine{
		Name:          resolver.raw.Name,
		sets:          resolver.sets,
		variables:     resolver.variables,
		initial:       initial,
		operations:    operations,
		currentName:   "current",
		stateSetName:  "STATE",
		transitionSet: "TRANSITION",
	}
	for _, operation := range operations {
		if operation.Name == "Advance" {
			machine.advance = operation
		}
	}
	return machine, resolver.diagnostics.Err()
}

func (resolver *resolver) resolveSets() {
	for _, set := range resolver.raw.Sets {
		values := make([]string, 0, len(set.Values))
		for _, item := range set.Values {
			values = append(values, item.Name)
			resolver.enums[item.Name] = set.Name
		}
		resolver.sets[set.Name] = values
	}
}

func (resolver *resolver) resolveVariables() {
	memberships := flattenMemberships(resolver.raw.Invariant)
	for _, membership := range memberships {
		identifier, ok := membership.Element.(rawIdentifierExpression)
		if !ok {
			continue
		}
		named, ok := membership.Set.(rawNamedSetExpression)
		if !ok {
			continue
		}
		kind := valueEnum
		if named.Name == "NAT" {
			kind = valueNat
		}
		if named.Name == "BOOL" {
			kind = valueBool
		}
		resolver.variables[identifier.Name] = variable{name: identifier.Name, kind: kind, set: named.Name}
	}
	if _, ok := resolver.variables["current"]; !ok {
		resolver.diagnostics = append(resolver.diagnostics, Diagnostic{Span: resolver.raw.Span, Message: "current variable must be declared in INVARIANT"})
	}
}

func (resolver *resolver) resolveInitialisation() map[string]value {
	values := map[string]value{}
	assignments := flattenAssignments(resolver.raw.Initialisation)
	for _, assignment := range assignments {
		variable, ok := resolver.variables[assignment.Name]
		if !ok {
			resolver.diagnostics = append(resolver.diagnostics, Diagnostic{Span: assignment.Span, Message: fmt.Sprintf("unknown variable %s", assignment.Name)})
			continue
		}
		values[assignment.Name] = resolver.resolveValue(variable, assignment.Value)
	}
	return values
}

func (resolver *resolver) resolveValue(variable variable, raw rawExpression) value {
	switch expression := raw.(type) {
	case rawNumberExpression:
		return value{kind: valueNat, nat: expression.Value}
	case rawIdentifierExpression:
		if variable.kind == valueBool {
			if expression.Name != "TRUE" && expression.Name != "FALSE" {
				resolver.diagnostics = append(resolver.diagnostics, Diagnostic{Span: raw.expressionSpan(), Message: fmt.Sprintf("invalid BOOL literal %s; expected TRUE or FALSE", expression.Name)})
			}
			return value{kind: valueBool, bool: expression.Name == "TRUE"}
		}
		return value{kind: valueEnum, enum: expression.Name}
	}
	resolver.diagnostics = append(resolver.diagnostics, Diagnostic{Span: raw.expressionSpan(), Message: fmt.Sprintf("unsupported value for %s", variable.name)})
	return value{}
}

func (resolver *resolver) resolveOperations() []operation {
	operations := []operation{}
	for _, rawOperation := range resolver.raw.Operations {
		operation := operation{Name: rawOperation.Name, Span: rawOperation.Span}
		for _, parameter := range rawOperation.Parameters {
			operation.Parameters = append(operation.Parameters, parameter.Name)
		}
		if rawOperation.Pre != nil {
			operation.Pre = resolver.resolvePredicate(rawOperation.Pre)
		}
		operation.Body = resolver.resolveSubstitution(rawOperation.Body)
		operations = append(operations, operation)
	}
	return operations
}

func (resolver *resolver) resolvePredicate(raw rawPredicate) predicate {
	switch predicate := raw.(type) {
	case rawBinaryPredicate:
		return binaryPredicate{operator: predicate.Operator, left: resolver.resolvePredicate(predicate.Left), right: resolver.resolvePredicate(predicate.Right)}
	case rawComparisonPredicate:
		return comparisonPredicate{operator: predicate.Operator, left: resolver.resolveExpression(predicate.Left), right: resolver.resolveExpression(predicate.Right)}
	case rawMembershipPredicate:
		return membershipPredicate{element: resolver.resolveExpression(predicate.Element), set: resolver.resolveSetExpression(predicate.Set)}
	}
	resolver.diagnostics = append(resolver.diagnostics, Diagnostic{Span: raw.predicateSpan(), Message: "unsupported predicate"})
	return comparisonPredicate{operator: "=", left: numberExpression{}, right: numberExpression{}}
}

func (resolver *resolver) resolveExpression(raw rawExpression) expression {
	switch expression := raw.(type) {
	case rawIdentifierExpression:
		return identifierExpression{name: expression.Name}
	case rawNumberExpression:
		return numberExpression{value: expression.Value}
	}
	resolver.diagnostics = append(resolver.diagnostics, Diagnostic{Span: raw.expressionSpan(), Message: "unsupported expression"})
	return numberExpression{}
}

func (resolver *resolver) resolveSetExpression(raw rawSetExpression) setExpression {
	switch rawExpression := raw.(type) {
	case rawNamedSetExpression:
		return namedSetExpression{name: rawExpression.Name, values: resolver.sets[rawExpression.Name]}
	case rawLiteralSetExpression:
		values := make([]expression, 0, len(rawExpression.Values))
		for _, item := range rawExpression.Values {
			values = append(values, resolver.resolveExpression(item))
		}
		return literalSetExpression{values: values}
	}
	resolver.diagnostics = append(resolver.diagnostics, Diagnostic{Span: raw.setExpressionSpan(), Message: "unsupported set expression"})
	return literalSetExpression{}
}

func (resolver *resolver) resolveSubstitution(raw rawSubstitution) substitution {
	switch rawSubstitution := raw.(type) {
	case rawAssignment:
		return assignment{name: rawSubstitution.Name, value: resolver.resolveExpression(rawSubstitution.Value)}
	case rawParallelAssignment:
		assignments := make([]assignment, 0, len(rawSubstitution.Assignments))
		for _, item := range rawSubstitution.Assignments {
			assignments = append(assignments, assignment{name: item.Name, value: resolver.resolveExpression(item.Value)})
		}
		return parallelAssignment{assignments: assignments}
	case rawIfSubstitution:
		branches := make([]guardedSubstitution, 0, len(rawSubstitution.Branches))
		for _, branch := range rawSubstitution.Branches {
			branches = append(branches, guardedSubstitution{condition: resolver.resolvePredicate(branch.Condition), body: resolver.resolveSubstitution(branch.Body)})
		}
		var fallback substitution
		if rawSubstitution.Else != nil {
			fallback = resolver.resolveSubstitution(rawSubstitution.Else)
		}
		return ifSubstitution{branches: branches, fallback: fallback}
	}
	resolver.diagnostics = append(resolver.diagnostics, Diagnostic{Span: raw.substitutionSpan(), Message: "unsupported substitution"})
	return parallelAssignment{}
}

func flattenMemberships(predicate rawPredicate) []rawMembershipPredicate {
	switch item := predicate.(type) {
	case rawMembershipPredicate:
		return []rawMembershipPredicate{item}
	case rawBinaryPredicate:
		left := flattenMemberships(item.Left)
		return append(left, flattenMemberships(item.Right)...)
	}
	return nil
}

func flattenAssignments(substitution rawSubstitution) []rawAssignment {
	switch item := substitution.(type) {
	case rawAssignment:
		return []rawAssignment{item}
	case rawParallelAssignment:
		return item.Assignments
	}
	return nil
}
