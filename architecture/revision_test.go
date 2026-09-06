package architecture_test

import (
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// Each registered domain record carries the expected revision consumed by the
// store's mandatory SQL compare-and-swap. Nested JSON values are not records.
func TestRegisteredDomainRecordsHaveOptimisticRevisions(t *testing.T) {
	t.Parallel()
	root := filepath.Join(repositoryRoot(t), "app", "domains")
	types := map[string]*ast.StructType{}
	registered := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		key := filepath.Dir(path) + "/"
		ast.Inspect(file, func(node ast.Node) bool {
			if declaration, ok := node.(*ast.TypeSpec); ok {
				if fields, ok := declaration.Type.(*ast.StructType); ok {
					types[key+declaration.Name.Name] = fields
				}
			}
			if call, ok := node.(*ast.CallExpr); ok {
				if selector, ok := call.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "Register" {
					for _, arg := range call.Args {
						if value, ok := arg.(*ast.CompositeLit); ok {
							if name, ok := value.Type.(*ast.Ident); ok {
								registered[key+name.Name] = true
							}
						}
					}
				}
			}
			return true
		})
		return nil
	})
	testutil.Ok(t, err)
	testutil.IsTrue(t, len(registered) >= 9)
	for name := range registered {
		fields := types[name]
		testutil.NotNil(t, fields)
		found := false
		for _, field := range fields.Fields.List {
			if field.Tag != nil {
				raw, err := strconv.Unquote(field.Tag.Value)
				testutil.Ok(t, err)
				if reflect.StructTag(raw).Get("store") == "revision" {
					found = true
				}
			}
		}
		testutil.ErrorIf(t, !found, "registered record %s lacks a SQL revision", name)
	}
}
