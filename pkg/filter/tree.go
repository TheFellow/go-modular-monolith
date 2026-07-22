package filter

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr/ast"
)

// Kind identifies an application-owned filter tree node.
type Kind uint8

const (
	KindLiteral Kind = iota + 1
	KindField
	KindList
	KindCall
	KindUnary
	KindBinary
)

// Node is the transport-neutral representation of a checked filter. Value is
// populated for literals, Name for fields/calls, Operator for unary/binary
// nodes, and Children for operands/arguments.
type Node struct {
	Kind     Kind
	Name     string
	Operator string
	Value    any
	Children []Node
}

func buildTree(node ast.Node) (Node, error) {
	switch n := node.(type) {
	case *ast.IdentifierNode:
		return Node{Kind: KindField, Name: n.Value}, nil
	case *ast.MemberNode:
		name, ok := fieldPath(n)
		if !ok {
			return Node{}, fmt.Errorf("only named field access is supported")
		}
		return Node{Kind: KindField, Name: name}, nil
	case *ast.StringNode:
		return Node{Kind: KindLiteral, Value: n.Value}, nil
	case *ast.IntegerNode:
		return Node{Kind: KindLiteral, Value: n.Value}, nil
	case *ast.FloatNode:
		return Node{Kind: KindLiteral, Value: n.Value}, nil
	case *ast.BoolNode:
		return Node{Kind: KindLiteral, Value: n.Value}, nil
	case *ast.NilNode:
		return Node{Kind: KindLiteral, Value: nil}, nil
	case *ast.ConstantNode:
		return Node{Kind: KindLiteral, Value: n.Value}, nil
	case *ast.ArrayNode:
		children, err := buildChildren(n.Nodes)
		return Node{Kind: KindList, Children: children}, err
	case *ast.UnaryNode:
		if n.Operator != "!" && n.Operator != "not" {
			return Node{}, fmt.Errorf("operator %q is not supported", n.Operator)
		}
		child, err := buildTree(n.Node)
		return Node{Kind: KindUnary, Operator: "!", Children: []Node{child}}, err
	case *ast.BinaryNode:
		if !isAllowedBinary(n.Operator) {
			return Node{}, fmt.Errorf("operator %q is not supported", n.Operator)
		}
		left, err := buildTree(n.Left)
		if err != nil {
			return Node{}, err
		}
		right, err := buildTree(n.Right)
		if err != nil {
			return Node{}, err
		}
		return Node{Kind: KindBinary, Operator: canonicalOperator(n.Operator), Children: []Node{left, right}}, nil
	case *ast.BuiltinNode:
		if n.Name != "date" && n.Name != "duration" {
			return Node{}, fmt.Errorf("function %q is not supported", n.Name)
		}
		children, err := buildChildren(n.Arguments)
		return Node{Kind: KindCall, Name: n.Name, Children: children}, err
	case *ast.CallNode:
		id, ok := n.Callee.(*ast.IdentifierNode)
		if !ok || (id.Value != "date" && id.Value != "duration") {
			return Node{}, fmt.Errorf("function calls other than date and duration are not supported")
		}
		children, err := buildChildren(n.Arguments)
		return Node{Kind: KindCall, Name: id.Value, Children: children}, err
	default:
		return Node{}, fmt.Errorf("expression construct %T is not supported", node)
	}
}

func buildChildren(nodes []ast.Node) ([]Node, error) {
	out := make([]Node, len(nodes))
	for i, node := range nodes {
		var err error
		out[i], err = buildTree(node)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func isAllowedBinary(op string) bool {
	switch op {
	case "&&", "and", "||", "or", "==", "!=", "<", "<=", ">", ">=", "in", "not in", "contains", "startsWith", "endsWith", "matches":
		return true
	default:
		return false
	}
}

func canonicalOperator(op string) string {
	switch op {
	case "and":
		return "&&"
	case "or":
		return "||"
	case "not":
		return "!"
	default:
		return op
	}
}

func fieldPath(node ast.Node) (string, bool) {
	switch n := node.(type) {
	case *ast.IdentifierNode:
		return n.Value, true
	case *ast.MemberNode:
		left, ok := fieldPath(n.Node)
		property, pok := n.Property.(*ast.StringNode)
		if !ok || !pok || n.Method {
			return "", false
		}
		return left + "." + property.Value, true
	default:
		return "", false
	}
}

func formatNode(node ast.Node, parentPrecedence int) string {
	switch n := node.(type) {
	case *ast.BinaryNode:
		op := canonicalOperator(n.Operator)
		if op == "contains" || op == "startsWith" || op == "endsWith" || op == "matches" {
			return formatNode(n.Left, 9) + "." + op + "(" + formatNode(n.Right, 0) + ")"
		}
		precedence := 5
		if op == "||" {
			precedence = 1
		}
		if op == "&&" {
			precedence = 2
		}
		s := formatNode(n.Left, precedence) + " " + op + " " + formatNode(n.Right, precedence+1)
		if precedence < parentPrecedence {
			return "(" + s + ")"
		}
		return s
	case *ast.UnaryNode:
		s := "!" + formatNode(n.Node, 8)
		if 8 < parentPrecedence {
			return "(" + s + ")"
		}
		return s
	case *ast.ArrayNode:
		parts := make([]string, len(n.Nodes))
		for i, child := range n.Nodes {
			parts[i] = formatNode(child, 0)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *ast.BuiltinNode:
		parts := make([]string, len(n.Arguments))
		for i, child := range n.Arguments {
			parts[i] = formatNode(child, 0)
		}
		return n.Name + "(" + strings.Join(parts, ", ") + ")"
	default:
		return node.String()
	}
}
