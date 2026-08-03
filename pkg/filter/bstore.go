package filter

import (
	"reflect"

	"github.com/mjl-/bstore"
)

// ApplyBstore adds an expression to a bstore query. Persisted-field predicates
// implied by the expression are pushed into native bstore filters; the complete
// expression is retained as FilterFn so every supported construct stays exact.
func ApplyBstore[Row, View any](q *bstore.Query[Row], expression *Expression[View], project func(Row) View) *bstore.Query[Row] {
	q = ApplyBstorePushdowns(q, expression)
	if expression == nil {
		return q
	}
	return q.FilterFn(func(row Row) bool {
		matched, err := expression.Match(project(row))
		if err != nil {
			// Parse only admits statically checked, non-failing constructs. A
			// runtime error is therefore a programmer/invariant failure and must
			// never be disguised as an ordinary non-match.
			panic(err)
		}
		return matched
	})
}

// ApplyBstorePushdowns adds only the safe persisted-field constraints from an
// expression to a bstore query. Callers use this staged form when the complete
// filter view depends on data that must be hydrated after rows are fetched.
// They must subsequently call Expression.Match with that complete view.
func ApplyBstorePushdowns[Row, View any](q *bstore.Query[Row], expression *Expression[View]) *bstore.Query[Row] {
	if expression == nil {
		return q
	}
	for _, p := range expression.pushdowns() {
		switch p.operator {
		case "==":
			q = q.FilterEqual(p.column, p.values...)
		case "!=":
			q = q.FilterNotEqual(p.column, p.values...)
		case ">":
			q = q.FilterGreater(p.column, p.values[0])
		case ">=":
			q = q.FilterGreaterEqual(p.column, p.values[0])
		case "<":
			q = q.FilterLess(p.column, p.values[0])
		case "<=":
			q = q.FilterLessEqual(p.column, p.values[0])
		case "in":
			q = q.FilterEqual(p.column, p.values...)
		case "not in":
			q = q.FilterNotEqual(p.column, p.values...)
		}
	}
	return q
}

type pushdown struct {
	column   string
	operator string
	values   []any
}

func (e *Expression[T]) pushdowns() []pushdown {
	return e.impliedPushdowns(e.tree, false)
}

// impliedPushdowns returns only predicates that must hold whenever node is
// true. Negation is carried down to leaves so we can recognize comparisons
// without changing the expression retained for exact residual evaluation.
func (e *Expression[T]) impliedPushdowns(node Node, negated bool) []pushdown {
	if node.Kind == KindUnary && node.Operator == "!" && len(node.Children) == 1 {
		return e.impliedPushdowns(node.Children[0], !negated)
	}
	if node.Kind == KindBinary && len(node.Children) == 2 && (node.Operator == "&&" || node.Operator == "||") {
		op := node.Operator
		if negated {
			if op == "&&" {
				op = "||"
			} else {
				op = "&&"
			}
		}
		left := e.impliedPushdowns(node.Children[0], negated)
		right := e.impliedPushdowns(node.Children[1], negated)
		if op == "&&" {
			return conjunctionPushdowns(left, right)
		}
		return disjunctionPushdowns(left, right)
	}
	if node.Kind == KindField {
		if p, ok := e.boolPushdown(node, !negated); ok {
			return []pushdown{p}
		}
		return nil
	}
	if p, ok := e.pushdown(node, negated); ok {
		return []pushdown{p}
	}
	return nil
}

func (e *Expression[T]) boolPushdown(node Node, value bool) (pushdown, bool) {
	field, ok := e.schema.field(node.Name)
	if !ok || field.Column == "" || field.Type.Kind() != reflect.Bool {
		return pushdown{}, false
	}
	converted, ok := convertLiteral(value, field.Type)
	if !ok {
		return pushdown{}, false
	}
	return pushdown{column: field.Column, operator: "==", values: []any{converted}}, true
}

func (e *Expression[T]) pushdown(node Node, negated bool) (pushdown, bool) {
	if node.Kind != KindBinary || len(node.Children) != 2 {
		return pushdown{}, false
	}
	op := node.Operator
	if negated {
		var ok bool
		op, ok = negateComparison(op)
		if !ok {
			return pushdown{}, false
		}
	}
	left, right := node.Children[0], node.Children[1]
	if left.Kind != KindField && right.Kind == KindField {
		left, right = right, left
		op = reverseComparison(op)
	}
	if left.Kind != KindField {
		return pushdown{}, false
	}
	field, ok := e.schema.field(left.Name)
	if !ok || field.Column == "" {
		return pushdown{}, false
	}
	var raw []any
	switch right.Kind {
	case KindLiteral:
		if op != "==" && op != "!=" && op != ">" && op != ">=" && op != "<" && op != "<=" {
			return pushdown{}, false
		}
		raw = []any{right.Value}
	case KindCall:
		if right.Value == nil || (op != "==" && op != "!=" && op != ">" && op != ">=" && op != "<" && op != "<=") {
			return pushdown{}, false
		}
		raw = []any{right.Value}
	case KindList:
		if op != "in" && op != "not in" {
			return pushdown{}, false
		}
		for _, item := range right.Children {
			if item.Kind != KindLiteral {
				return pushdown{}, false
			}
			raw = append(raw, item.Value)
		}
	case KindField, KindUnary, KindBinary:
		return pushdown{}, false
	}
	values := make([]any, len(raw))
	for i, value := range raw {
		converted, ok := convertLiteral(value, field.Type)
		if !ok {
			return pushdown{}, false
		}
		values[i] = converted
	}
	switch op {
	case "in":
		op = "=="
	case "not in":
		op = "!="
	}
	if op == "==" || op == "!=" {
		values = appendUniqueValues(nil, values...)
	}
	return pushdown{column: field.Column, operator: op, values: values}, len(values) > 0
}

func conjunctionPushdowns(left, right []pushdown) []pushdown {
	out := make([]pushdown, 0, len(left)+len(right))
	for _, p := range append(append([]pushdown{}, left...), right...) {
		if p.operator == "!=" {
			merged := false
			for i := range out {
				if out[i].operator == "!=" && out[i].column == p.column && compatibleValueTypes(out[i].values, p.values) {
					out[i].values = appendUniqueValues(out[i].values, p.values...)
					merged = true
					break
				}
			}
			if merged {
				continue
			}
		}
		out = appendUniquePushdown(out, p)
	}
	return out
}

// disjunctionPushdowns keeps predicates required by both alternatives. Equal
// constraints for the same column can also be safely widened into one
// multi-value equality: either branch implies the stored value is in the
// combined set. Other differing predicates cannot be pushed across OR.
func disjunctionPushdowns(left, right []pushdown) []pushdown {
	var out []pushdown
	for _, lp := range left {
		for _, rp := range right {
			if equivalentPushdown(lp, rp) {
				out = appendUniquePushdown(out, lp)
				break
			}
		}
	}

	seenColumns := map[string]bool{}
	for _, lp := range left {
		if lp.operator != "==" || seenColumns[lp.column] {
			continue
		}
		leftValues, leftOK := equalityValues(left, lp.column)
		rightValues, rightOK := equalityValues(right, lp.column)
		if !leftOK || !rightOK || !compatibleValueTypes(leftValues, rightValues) {
			continue
		}
		values := appendUniqueValues(leftValues, rightValues...)
		out = appendUniquePushdown(out, pushdown{column: lp.column, operator: "==", values: values})
		seenColumns[lp.column] = true
	}
	return out
}

func equalityValues(pushdowns []pushdown, column string) ([]any, bool) {
	var values []any
	for _, p := range pushdowns {
		if p.operator != "==" || p.column != column {
			continue
		}
		if len(values) > 0 && !compatibleValueTypes(values, p.values) {
			return nil, false
		}
		values = appendUniqueValues(values, p.values...)
	}
	return values, len(values) > 0
}

func appendUniquePushdown(pushdowns []pushdown, candidate pushdown) []pushdown {
	for _, p := range pushdowns {
		if equivalentPushdown(p, candidate) {
			return pushdowns
		}
	}
	return append(pushdowns, candidate)
}

func equivalentPushdown(a, b pushdown) bool {
	if a.column != b.column || a.operator != b.operator || len(a.values) != len(b.values) {
		return false
	}
	if a.operator != "==" && a.operator != "!=" {
		return reflect.DeepEqual(a.values, b.values)
	}
	for _, av := range a.values {
		found := false
		for _, bv := range b.values {
			if reflect.DeepEqual(av, bv) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func appendUniqueValues(values []any, candidates ...any) []any {
	for _, candidate := range candidates {
		found := false
		for _, value := range values {
			if reflect.DeepEqual(value, candidate) {
				found = true
				break
			}
		}
		if !found {
			values = append(values, candidate)
		}
	}
	return values
}

func compatibleValueTypes(a, b []any) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	t := reflect.TypeOf(a[0])
	for _, values := range [][]any{a, b} {
		for _, value := range values {
			if reflect.TypeOf(value) != t {
				return false
			}
		}
	}
	return true
}

func negateComparison(op string) (string, bool) {
	switch op {
	case "==":
		return "!=", true
	case "!=":
		return "==", true
	case ">":
		return "<=", true
	case ">=":
		return "<", true
	case "<":
		return ">=", true
	case "<=":
		return ">", true
	case "in":
		return "not in", true
	case "not in":
		return "in", true
	default:
		return "", false
	}
}

func reverseComparison(op string) string {
	switch op {
	case ">":
		return "<"
	case ">=":
		return "<="
	case "<":
		return ">"
	case "<=":
		return ">="
	default:
		return op
	}
}

func convertLiteral(value any, target reflect.Type) (any, bool) {
	if value == nil || target.Kind() == reflect.Pointer {
		return nil, false
	}
	v := reflect.ValueOf(value)
	if v.Type().AssignableTo(target) {
		return value, true
	}
	if !v.Type().ConvertibleTo(target) {
		return nil, false
	}
	switch target.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return v.Convert(target).Interface(), true
	case reflect.Invalid, reflect.Uintptr, reflect.Complex64, reflect.Complex128,
		reflect.Array, reflect.Chan, reflect.Func, reflect.Interface,
		reflect.Map, reflect.Pointer, reflect.Slice, reflect.Struct,
		reflect.UnsafePointer:
		return nil, false
	}
	return nil, false
}
