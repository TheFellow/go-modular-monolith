package architecture_test

import (
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// TestEveryDomainIsComposed catches the easy-to-miss case where a new domain
// package is added but never exposed or initialized by the application root.
// The domain directories remain the source of truth; this test deliberately
// does not introduce a second registration manifest.
func TestEveryDomainIsComposed(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	domainEntries, err := os.ReadDir(filepath.Join(root, "app", "domains"))
	testutil.ErrorIf(t, err != nil, "read domain directories: %v", err)

	appFile, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "app", "app.go"), nil, 0)
	testutil.ErrorIf(t, err != nil, "parse application composition: %v", err)

	domainAliases := make(map[string]string)
	for _, spec := range appFile.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		testutil.ErrorIf(t, err != nil, "parse import path %s: %v", spec.Path.Value, err)
		const domainPrefix = "github.com/TheFellow/go-modular-monolith/app/domains/"
		domain, ok := cutExactChild(path, domainPrefix)
		if !ok {
			continue
		}
		alias := domain
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		domainAliases[domain] = alias
	}

	moduleFields := appModuleFields(t, appFile)
	initializedFields := initializedAppFields(t, appFile)
	for _, entry := range domainEntries {
		if !entry.IsDir() {
			continue
		}
		domain := entry.Name()
		alias, imported := domainAliases[domain]
		if !imported {
			t.Errorf("domain %q is not imported by app/app.go", domain)
			continue
		}
		field, exposed := moduleFields[alias]
		if !exposed {
			t.Errorf("domain %q is not exposed as a *%s.Module field on App", domain, alias)
			continue
		}
		if !initializedFields[field] {
			t.Errorf("domain %q App field %q is not initialized by New", domain, field)
		}
	}
}

func cutExactChild(path, prefix string) (string, bool) {
	if len(path) <= len(prefix) || path[:len(prefix)] != prefix {
		return "", false
	}
	child := path[len(prefix):]
	for _, r := range child {
		if r == '/' {
			return "", false
		}
	}
	return child, true
}

func appModuleFields(t *testing.T, file *ast.File) map[string]string {
	t.Helper()
	fields := make(map[string]string)
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, spec := range general.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "App" {
				continue
			}
			structure, ok := typeSpec.Type.(*ast.StructType)
			testutil.ErrorIf(t, !ok, "%v", "App is not a struct")
			for _, field := range structure.Fields.List {
				pointer, ok := field.Type.(*ast.StarExpr)
				if !ok {
					continue
				}
				selector, ok := pointer.X.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "Module" || len(field.Names) != 1 {
					continue
				}
				alias, ok := selector.X.(*ast.Ident)
				if ok {
					fields[alias.Name] = field.Names[0].Name
				}
			}
			return fields
		}
	}
	testutil.Fail(t, "%v", "App struct not found")
	return nil
}

func initializedAppFields(t *testing.T, file *ast.File) map[string]bool {
	t.Helper()
	fields := make(map[string]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		appType, ok := literal.Type.(*ast.Ident)
		if !ok || appType.Name != "App" {
			return true
		}
		for _, element := range literal.Elts {
			pair, ok := element.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := pair.Key.(*ast.Ident)
			if !ok {
				continue
			}
			if value, nilValue := pair.Value.(*ast.Ident); !nilValue || value.Name != "nil" {
				fields[key.Name] = true
			}
		}
		return true
	})
	testutil.ErrorIf(t, len(fields) == 0, "%v", "initialized App literal not found")
	return fields
}
