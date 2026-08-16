package circuitb

import "fmt"

func (machine Machine) State(facts map[string]value) (StateReport, error) {
	return machine.StateAt("", facts)
}

func (machine Machine) StateAt(current string, facts map[string]value) (StateReport, error) {
	values := machine.valuesWithFacts(facts)
	return machine.stateFromValues(current, values)
}

func (machine Machine) StateAtWithBooleans(current string, booleans map[string]bool) (StateReport, error) {
	values := machine.valuesWithBooleans(booleans)
	return machine.stateFromValues(current, values)
}

func (machine Machine) stateFromValues(current string, values map[string]value) (StateReport, error) {
	if current != "" {
		values[machine.currentName] = value{kind: valueEnum, enum: current}
	}
	state := values[machine.currentName].enum
	report := StateReport{Current: state}
	for _, transition := range machine.sets[machine.transitionSet] {
		bindings := map[string]value{"evt": {kind: valueEnum, enum: transition}}
		failed := machine.advance.Pre.explain(values, bindings)
		status := CallStatus{Call: fmt.Sprintf("Advance(%s)", transition), Failed: failed}
		if len(failed) == 0 {
			report.Enabled = append(report.Enabled, status)
		} else {
			report.Blocked = append(report.Blocked, status)
		}
	}
	return report, nil
}

func (machine Machine) Advance(event string, facts map[string]value) (AdvanceResult, error) {
	return machine.AdvanceFrom(event, "", facts)
}

func (machine Machine) AdvanceFrom(event string, current string, facts map[string]value) (AdvanceResult, error) {
	values := machine.valuesWithFacts(facts)
	return machine.advanceFromValues(event, current, values)
}

func (machine Machine) AdvanceFromWithBooleans(event string, current string, booleans map[string]bool) (AdvanceResult, error) {
	values := machine.valuesWithBooleans(booleans)
	return machine.advanceFromValues(event, current, values)
}

func (machine Machine) advanceFromValues(event string, current string, values map[string]value) (AdvanceResult, error) {
	if current != "" {
		values[machine.currentName] = value{kind: valueEnum, enum: current}
	}
	from := values[machine.currentName].enum
	bindings := map[string]value{"evt": {kind: valueEnum, enum: event}}
	failed := machine.advance.Pre.explain(values, bindings)
	if len(failed) > 0 {
		return AdvanceResult{Allowed: false, From: from, Failed: failed}, nil
	}
	next := machine.advance.Body.apply(values, bindings)
	return AdvanceResult{Allowed: true, From: from, To: next[machine.currentName].enum}, nil
}

func (machine Machine) valuesWithBooleans(booleans map[string]bool) map[string]value {
	facts := map[string]value{}
	for key, item := range booleans {
		facts[key] = value{kind: valueBool, bool: item}
	}
	return machine.valuesWithFacts(facts)
}

func (machine Machine) valuesWithFacts(facts map[string]value) map[string]value {
	values := map[string]value{}
	for key, item := range machine.initial {
		values[key] = item
	}
	for key, item := range facts {
		values[key] = item
	}
	return values
}

func (predicate binaryPredicate) evaluate(values map[string]value, bindings map[string]value) bool {
	if predicate.operator == "or" {
		return predicate.left.evaluate(values, bindings) || predicate.right.evaluate(values, bindings)
	}
	return predicate.left.evaluate(values, bindings) && predicate.right.evaluate(values, bindings)
}

func (predicate binaryPredicate) explain(values map[string]value, bindings map[string]value) []string {
	if predicate.evaluate(values, bindings) {
		return nil
	}
	if predicate.operator == "or" {
		left := predicate.left.explain(values, bindings)
		right := predicate.right.explain(values, bindings)
		if len(left) <= len(right) {
			return left
		}
		return right
	}
	failed := predicate.left.explain(values, bindings)
	failed = append(failed, predicate.right.explain(values, bindings)...)
	return failed
}

func (predicate comparisonPredicate) evaluate(values map[string]value, bindings map[string]value) bool {
	left := predicate.left.evaluate(values, bindings)
	right := predicate.right.evaluate(values, bindings)
	switch predicate.operator {
	case "=":
		return left == right
	case "/=":
		return left != right
	case "<":
		return left.nat < right.nat
	case "<=":
		return left.nat <= right.nat
	case ">":
		return left.nat > right.nat
	case ">=":
		return left.nat >= right.nat
	}
	return false
}

func (predicate comparisonPredicate) explain(values map[string]value, bindings map[string]value) []string {
	if predicate.evaluate(values, bindings) {
		return nil
	}
	return []string{predicate.explainText(values, bindings)}
}

func (predicate membershipPredicate) evaluate(values map[string]value, bindings map[string]value) bool {
	return predicate.set.contains(predicate.element.evaluate(values, bindings), values, bindings)
}

func (predicate membershipPredicate) explain(values map[string]value, bindings map[string]value) []string {
	if predicate.evaluate(values, bindings) {
		return nil
	}
	return []string{predicate.explainText(values, bindings)}
}

func (predicate comparisonPredicate) explainText(_ map[string]value, _ map[string]value) string {
	return predicate.left.format() + " " + predicate.operator + " " + predicate.right.format()
}

func (predicate membershipPredicate) explainText(_ map[string]value, _ map[string]value) string {
	return predicate.element.format() + " : " + predicate.set.format()
}

func (predicate notPredicate) evaluate(values map[string]value, bindings map[string]value) bool {
	return !predicate.inner.evaluate(values, bindings)
}

func (predicate notPredicate) explain(values map[string]value, bindings map[string]value) []string {
	if predicate.evaluate(values, bindings) {
		return nil
	}
	return []string{"not(" + describePredicate(predicate.inner) + ")"}
}

func describePredicate(predicate predicate) string {
	switch inner := predicate.(type) {
	case comparisonPredicate:
		return inner.left.format() + " " + inner.operator + " " + inner.right.format()
	case membershipPredicate:
		return inner.element.format() + " : " + inner.set.format()
	case notPredicate:
		return "not(" + describePredicate(inner.inner) + ")"
	case binaryPredicate:
		return describePredicate(inner.left) + " " + inner.operator + " " + describePredicate(inner.right)
	}
	return "predicate"
}

func (expression identifierExpression) evaluate(values map[string]value, bindings map[string]value) value {
	if item, ok := bindings[expression.name]; ok {
		return item
	}
	if item, ok := values[expression.name]; ok {
		return item
	}
	if expression.name == booleanLiteralTrue {
		return value{kind: valueBool, bool: true}
	}
	if expression.name == booleanLiteralFalse {
		return value{kind: valueBool, bool: false}
	}
	return value{kind: valueEnum, enum: expression.name}
}

func (expression identifierExpression) format() string {
	return expression.name
}

func (expression numberExpression) evaluate(_ map[string]value, _ map[string]value) value {
	return value{kind: valueNat, nat: expression.value}
}

func (expression numberExpression) format() string {
	return fmt.Sprintf("%d", expression.value)
}

func (expression namedSetExpression) contains(candidate value, _ map[string]value, _ map[string]value) bool {
	for _, item := range expression.values {
		if candidate.kind == valueEnum && candidate.enum == item {
			return true
		}
	}
	return (expression.name == "NAT" && candidate.kind == valueNat) || (expression.name == "BOOL" && candidate.kind == valueBool)
}

func (expression namedSetExpression) format() string {
	return expression.name
}

func (expression literalSetExpression) contains(candidate value, values map[string]value, bindings map[string]value) bool {
	for _, item := range expression.values {
		if candidate == item.evaluate(values, bindings) {
			return true
		}
	}
	return false
}

func (expression literalSetExpression) format() string {
	return "set literal"
}

func (substitution assignment) apply(values map[string]value, bindings map[string]value) map[string]value {
	next := cloneValues(values)
	next[substitution.name] = substitution.value.evaluate(values, bindings)
	return next
}

func (substitution parallelAssignment) apply(values map[string]value, bindings map[string]value) map[string]value {
	next := cloneValues(values)
	updates := map[string]value{}
	for _, item := range substitution.assignments {
		updates[item.name] = item.value.evaluate(values, bindings)
	}
	for key, item := range updates {
		next[key] = item
	}
	return next
}

func (substitution ifSubstitution) apply(values map[string]value, bindings map[string]value) map[string]value {
	for _, branch := range substitution.branches {
		if branch.condition.evaluate(values, bindings) {
			return branch.body.apply(values, bindings)
		}
	}
	if substitution.fallback != nil {
		return substitution.fallback.apply(values, bindings)
	}
	return cloneValues(values)
}

func cloneValues(values map[string]value) map[string]value {
	clone := map[string]value{}
	for key, item := range values {
		clone[key] = item
	}
	return clone
}
