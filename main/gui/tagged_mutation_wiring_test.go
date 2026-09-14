package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

// TestFyneMutationWorkflowsPassTagsToOwningCommand is an executable wiring
// contract for the fifteen mutation paths exposed with complete-set tags.
func TestFyneMutationWorkflowsPassTagsToOwningCommand(t *testing.T) {
	t.Parallel()
	type workflow struct {
		domain, method string
		callbacks      []string
	}
	workflows := []workflow{
		{"drinks", "Save", []string{"Create", "Update"}},
		{"ingredients", "Submit", []string{"Create", "Update"}},
		{"inventory", "Submit", []string{"Adjust", "Set"}},
		{"menus", "Save", []string{"Create", "Update"}},
		{"menus", "AddDrink", []string{"AddDrink"}},
		{"menus", "RemoveDrink", []string{"RemoveDrink"}},
		{"menus", "Publish", []string{"Publish"}},
		{"menus", "ReturnToDraft", []string{"Draft"}},
		{"orders", "SavePlace", []string{"Place"}},
		{"orders", "confirm", []string{"Cancel", "Complete"}},
	}
	total := 0
	for _, workflow := range workflows {
		path := filepath.Join("..", "..", "app", "domains", workflow.domain, "surfaces", "gui", "presenter.go")
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		testutil.Ok(t, err)
		calls := 0
		callbacks := make([]string, 0, len(workflow.callbacks))
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Name.Name != workflow.method || function.Recv == nil {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if slices.Contains(workflow.callbacks, selector.Sel.Name) {
					for _, arg := range call.Args {
						edit, ok := arg.(*ast.CallExpr)
						if !ok {
							continue
						}
						selected, ok := edit.Fun.(*ast.SelectorExpr)
						if ok && selected.Sel.Name == "Replace" {
							calls++
							callbacks = append(callbacks, selector.Sel.Name)
						}
					}
				}
				return true
			})
		}
		slices.Sort(callbacks)
		expected := slices.Clone(workflow.callbacks)
		slices.Sort(expected)
		testutil.ErrorIf(t, calls != len(expected) || !slices.Equal(callbacks, expected), "%s.%s wires %d domain commands with tags for %v, want %v", workflow.domain, workflow.method, calls, callbacks, expected)
		total += calls
	}
	testutil.Equals(t, total, 15)
}
