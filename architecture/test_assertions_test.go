package architecture_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestTestsUseTestutilAssertions(t *testing.T) {
	root := repositoryRoot(t)
	files := token.NewFileSet()
	var violations []string

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, parseErr := parser.ParseFile(files, path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (selector.Sel.Name != "Fatal" && selector.Sel.Name != "Fatalf" && selector.Sel.Name != "FailNow") {
				return true
			}
			position := files.Position(call.Pos())
			relative, relErr := filepath.Rel(root, position.Filename)
			if relErr != nil {
				relative = position.Filename
			}
			violations = append(violations, fmt.Sprintf("%s:%d uses %s; use testutil instead", relative, position.Line, selector.Sel.Name))
			return true
		})
		return nil
	})

	testutil.Ok(t, err)
	testutil.Equals(t, violations, []string(nil))
}
