// Package filter provides typed, reusable filter expressions for application
// list operations. Expressions use Expr syntax and are compiled against a
// concrete view type, so misspelled fields and incompatible comparisons fail
// before a query is executed.
package filter

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/vm"

	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
)

// Field describes one user-visible field in a filter schema.
type Field struct {
	Name        string
	Type        reflect.Type
	Description string
	Column      string
}

// Schema is the typed contract accepted by a list operation.
type Schema[T any] struct {
	fields   []Field
	examples []string
}

// NewSchema builds a schema from exported fields tagged with expr. A filter
// tag supplies help text and filter-column opts a top-level field into bstore
// pushdown.
func NewSchema[T any](examples ...string) Schema[T] {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		panic("filter: schema type must be concrete")
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		panic("filter: schema type must be a struct")
	}
	return Schema[T]{fields: collectFields(t, ""), examples: slices.Clone(examples)}
}

func collectFields(t reflect.Type, prefix string) []Field {
	var out []Field
	for i := range t.NumField() {
		f := t.Field(i)
		name := f.Tag.Get("expr")
		if name == "-" || !f.IsExported() {
			continue
		}
		if name == "" {
			name = f.Name
		}
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		description := f.Tag.Get("filter")
		if description != "" || f.Tag.Get("filter-column") != "" {
			out = append(out, Field{Name: path, Type: f.Type, Description: description, Column: f.Tag.Get("filter-column")})
		}
		ft := f.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct && ft.PkgPath() != "time" {
			out = append(out, collectFields(ft, path)...)
		}
	}
	return out
}

func (s Schema[T]) Fields() []Field    { return slices.Clone(s.fields) }
func (s Schema[T]) Examples() []string { return slices.Clone(s.examples) }
func (s Schema[T]) field(name string) (Field, bool) {
	for _, f := range s.fields {
		if f.Name == name {
			return f, true
		}
	}
	return Field{}, false
}

// Expression is a checked filter and its application-owned syntax tree.
type Expression[T any] struct {
	source    string
	canonical string
	schema    Schema[T]
	program   *vm.Program
	tree      Node
}

func (e *Expression[T]) Source() string { return e.source }
func (e *Expression[T]) String() string { return e.canonical }
func (e *Expression[T]) Tree() Node     { return e.tree }

// Parse compiles source against schema. Empty input means no expression.
func Parse[T any](schema Schema[T], source string) (*Expression[T], error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, nil
	}
	var zero T
	program, err := expr.Compile(source,
		expr.Env(zero),
		expr.AsBool(),
		expr.Optimize(false),
		expr.Patch(dotPredicatePatcher{}),
	)
	if err != nil {
		return nil, apperrors.Invalidf("invalid filter: %w", err)
	}
	tree, err := buildTree(program.Node())
	if err != nil {
		return nil, apperrors.Invalidf("invalid filter: %w", err)
	}
	return &Expression[T]{source: source, canonical: formatNode(program.Node(), 0), schema: schema, program: program, tree: tree}, nil
}

// Match evaluates the expression against one typed filter view.
func (e *Expression[T]) Match(value T) (bool, error) {
	if e == nil {
		return true, nil
	}
	result, err := expr.Run(e.program, value)
	if err != nil {
		return false, fmt.Errorf("evaluate filter: %w", err)
	}
	matched, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("evaluate filter: expression returned %T", result)
	}
	return matched, nil
}

type dotPredicatePatcher struct{}

func (dotPredicatePatcher) Visit(node *ast.Node) {
	call, ok := (*node).(*ast.CallNode)
	if !ok || len(call.Arguments) != 1 {
		return
	}
	member, ok := call.Callee.(*ast.MemberNode)
	if !ok || !member.Method {
		return
	}
	property, ok := member.Property.(*ast.StringNode)
	if !ok || !slices.Contains([]string{"contains", "startsWith", "endsWith", "matches"}, property.Value) {
		return
	}
	ast.Patch(node, &ast.BinaryNode{Operator: property.Value, Left: member.Node, Right: call.Arguments[0]})
}
