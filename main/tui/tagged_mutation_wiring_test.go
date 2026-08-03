package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestTUIMutationWorkflowsUseAtomicTagComposition(t *testing.T) {
	t.Parallel()
	type workflow struct{ domain, file, function, action, callback string }
	workflows := []workflow{
		{"drinks", "create_vm.go", "submit", "create", "Create"},
		{"drinks", "edit_vm.go", "submit", "update", "Update"},
		{"ingredients", "create_vm.go", "submit", "create", "Create"},
		{"ingredients", "edit_vm.go", "submit", "update", "Update"},
		{"inventory", "adjust_vm.go", "submit", "adjust", "Adjust"},
		{"inventory", "set_vm.go", "submit", "set", "Set"},
		{"menus", "create_vm.go", "submit", "create", "Create"},
		{"menus", "rename_vm.go", "submit", "update", "Update"},
		{"menus", "list_vm.go", "Update", "add-drink", "AddDrink"},
		{"menus", "list_vm.go", "performRemoveDrink", "remove-drink", "RemoveDrink"},
		{"menus", "list_vm.go", "performPublish", "publish", "Publish"},
		{"menus", "list_vm.go", "performDraft", "draft", "Draft"},
		{"orders", "place_vm.go", "submit", "place", "Place"},
		{"orders", "list_vm.go", "performComplete", "complete", "Complete"},
		{"orders", "list_vm.go", "performCancel", "cancel", "Cancel"},
	}
	for _, workflow := range workflows {
		path := filepath.Join("..", "..", "app", "domains", workflow.domain, "surfaces", "tui", workflow.file)
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		testutil.Ok(t, err)
		calls := 0
		callbacks := 0
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Name.Name != workflow.function || function.Recv == nil {
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
				pkg, appCall := selector.X.(*ast.Ident)
				if appCall && pkg.Name == "app" && selector.Sel.Name == "RunTaggedMutation" {
					calls++
					for _, arg := range call.Args {
						ast.Inspect(arg, func(node ast.Node) bool {
							invoked, found := node.(*ast.SelectorExpr)
							if found && invoked.Sel.Name == workflow.callback {
								callbacks++
							}
							return true
						})
					}
				}
				return true
			})
		}
		testutil.ErrorIf(t, calls != 1 || callbacks != 1, "%s %s wiring has %d atomic tagged mutations around %d %s callbacks, want 1 and 1", workflow.domain, workflow.action, calls, callbacks, workflow.callback)
	}
}
