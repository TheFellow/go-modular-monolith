package filter

import (
	"reflect"

	"github.com/mjl-/bstore"
)

// ApplyBstore adds an expression to a bstore query. Required top-level
// comparisons on explicitly mapped fields are pushed into native bstore
// filters; the complete expression is retained as FilterFn so arbitrary OR,
// NOT, parentheses, derived fields, and string predicates stay exact.
func ApplyBstore[Row, View any](q *bstore.Query[Row], expression *Expression[View], project func(Row) View) *bstore.Query[Row] {
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

type pushdown struct {
	column   string
	operator string
	values   []any
}

func (e *Expression[T]) pushdowns() []pushdown {
	var out []pushdown
	var visit func(Node)
	visit = func(node Node) {
		if node.Kind == KindBinary && node.Operator == "&&" {
			visit(node.Children[0])
			visit(node.Children[1])
			return
		}
		if p, ok := e.pushdown(node); ok {
			out = append(out, p)
		}
	}
	visit(e.tree)
	return out
}

func (e *Expression[T]) pushdown(node Node) (pushdown, bool) {
	if node.Kind != KindBinary || len(node.Children) != 2 {
		return pushdown{}, false
	}
	op := node.Operator
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
		raw = []any{right.Value}
	case KindCall:
		if right.Value == nil {
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
	return pushdown{column: field.Column, operator: op, values: values}, len(values) > 0
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
