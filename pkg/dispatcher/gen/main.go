package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"
)

//go:embed dispatcher.go.tpl
var dispatcherTemplate string

type eventType struct {
	PkgPath string
	Name    string
}

type handlerType struct {
	PkgPath      string
	Name         string
	EventPkgPath string
	EventName    string
	HasHandling  bool
	VarName      string
}

type eventGroup struct {
	Event    eventType
	Handlers []handlerType
}

func main() {
	repoRoot := filepath.Clean(filepath.Join(".", "..", ".."))

	modulePath, err := readModulePath(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		fatalf("read module path: %v", err)
	}

	events, err := scanEvents(repoRoot, modulePath)
	if err != nil {
		fatalf("scan events: %v", err)
	}

	handlers, err := scanHandlers(repoRoot, modulePath)
	if err != nil {
		fatalf("scan handlers: %v", err)
	}

	eventIndex := make(map[string]eventType, len(events))
	for _, e := range events {
		eventIndex[e.PkgPath+"."+e.Name] = e
	}

	matched := make([]handlerType, 0, len(handlers))
	for _, h := range handlers {
		if _, ok := eventIndex[h.EventPkgPath+"."+h.EventName]; ok {
			matched = append(matched, h)
		}
	}

	sort.Slice(matched, func(i, j int) bool {
		if matched[i].EventPkgPath != matched[j].EventPkgPath {
			return matched[i].EventPkgPath < matched[j].EventPkgPath
		}
		if matched[i].EventName != matched[j].EventName {
			return matched[i].EventName < matched[j].EventName
		}
		if matched[i].PkgPath != matched[j].PkgPath {
			return matched[i].PkgPath < matched[j].PkgPath
		}
		return matched[i].Name < matched[j].Name
	})

	groupIndex := make(map[string]*eventGroup)
	groups := make([]*eventGroup, 0)

	for _, h := range matched {
		key := h.EventPkgPath + "." + h.EventName
		g, ok := groupIndex[key]
		if !ok {
			ev := eventIndex[key]
			g = &eventGroup{Event: ev}
			groupIndex[key] = g
			groups = append(groups, g)
		}
		g.Handlers = append(g.Handlers, h)
	}

	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Event.PkgPath != groups[j].Event.PkgPath {
			return groups[i].Event.PkgPath < groups[j].Event.PkgPath
		}
		return groups[i].Event.Name < groups[j].Event.Name
	})

	assignHandlerVarNames(groups)

	type importSpec struct {
		Alias string
		Path  string
	}

	importAlias := map[string]string{}
	nextAlias := func(path string) string {
		if a, ok := importAlias[path]; ok {
			return a
		}

		alias := defaultAlias(modulePath, path)
		if alias == "" {
			alias = "pkg"
		}

		used := map[string]struct{}{}
		for _, a := range importAlias {
			used[a] = struct{}{}
		}

		if _, ok := used[alias]; ok {
			base := alias
			for i := 2; ; i++ {
				candidate := base + strconv.Itoa(i)
				if _, ok := used[candidate]; !ok {
					alias = candidate
					break
				}
			}
		}

		importAlias[path] = alias
		return alias
	}

	imports := make([]importSpec, 0)
	addImport := func(path string) string {
		alias := nextAlias(path)
		imports = append(imports, importSpec{Alias: alias, Path: path})
		return alias
	}

	middlewareImportPath := modulePath + "/pkg/middleware"
	middlewareAlias := addImport(middlewareImportPath)

	// Add event + handler package imports.
	for _, g := range groups {
		addImport(g.Event.PkgPath)
		for _, h := range g.Handlers {
			addImport(h.PkgPath)
		}
	}

	// De-dup and sort imports by path for determinism.
	uniq := map[string]importSpec{}
	for _, imp := range imports {
		uniq[imp.Path] = imp
	}
	imports = imports[:0]
	for _, imp := range uniq {
		imports = append(imports, imp)
	}
	sort.Slice(imports, func(i, j int) bool { return imports[i].Path < imports[j].Path })

	tmpl := template.Must(template.New("dispatcher").Parse(dispatcherTemplate))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{
		"Imports":         imports,
		"Groups":          groups,
		"ImportAlias":     importAlias,
		"MiddlewareAlias": middlewareAlias,
	}); err != nil {
		fatalf("execute template: %v", err)
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		fatalf("format source: %v\n\n%s", err, buf.String())
	}

	outPath := filepath.Join("dispatcher_gen.go")
	if err := os.WriteFile(outPath, src, 0o644); err != nil {
		fatalf("write %s: %v", outPath, err)
	}
}

func assignHandlerVarNames(groups []*eventGroup) {
	for _, g := range groups {
		used := map[string]int{}
		for i := range g.Handlers {
			base := domainFromPkgPath(g.Handlers[i].PkgPath)
			if base == "" {
				base = "handler"
			}
			base = base + "Handler"
			if token.Lookup(base).IsKeyword() {
				base = base + "H"
			}

			used[base]++
			name := base
			if used[base] > 1 {
				name = fmt.Sprintf("%s%d", base, used[base])
			}
			g.Handlers[i].VarName = name
		}
	}
}

func domainFromPkgPath(pkgPath string) string {
	if pkgPath == "" {
		return ""
	}

	parts := strings.Split(pkgPath, "/")
	for i, part := range parts {
		if part == "domains" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	for i, part := range parts {
		if part == "handlers" && i > 0 {
			return parts[i-1]
		}
	}

	return ""
}

func lowerCamel(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)

	// Find leading uppercase runes.
	i := 0
	for i < len(r) && unicode.IsUpper(r[i]) {
		i++
	}
	if i == 0 {
		return s
	}
	// All uppercase: lower all.
	if i == len(r) {
		for j := range r {
			r[j] = unicode.ToLower(r[j])
		}
		return string(r)
	}
	// Single leading uppercase: lower it.
	if i == 1 {
		r[0] = unicode.ToLower(r[0])
		return string(r)
	}

	// Multiple leading uppercase, followed by a lowercase. Lowercase all but the last
	// leading uppercase rune, e.g. HTTPServer -> httpServer.
	for j := 0; j < i-1; j++ {
		r[j] = unicode.ToLower(r[j])
	}
	return string(r)
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func readModulePath(goModPath string) (string, error) {
	b, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("module directive not found")
}

func scanEvents(repoRoot, modulePath string) ([]eventType, error) {
	var out []eventType

	fset := token.NewFileSet()

	for _, root := range []string{
		filepath.Join(repoRoot, "app"),
		filepath.Join(repoRoot, "pkg"),
	} {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if !strings.Contains(filepath.ToSlash(path), "/events/") {
				return nil
			}

			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return fmt.Errorf("parse %s: %w", path, err)
			}

			dir := filepath.Dir(path)
			relDir, err := filepath.Rel(repoRoot, dir)
			if err != nil {
				return err
			}
			pkgPath := modulePath + "/" + filepath.ToSlash(relDir)

			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.TYPE {
					continue
				}
				for _, spec := range gen.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if _, ok := ts.Type.(*ast.StructType); !ok {
						continue
					}
					out = append(out, eventType{PkgPath: pkgPath, Name: ts.Name.Name})
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].PkgPath != out[j].PkgPath {
			return out[i].PkgPath < out[j].PkgPath
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func scanHandlers(repoRoot, modulePath string) ([]handlerType, error) {
	type handlerKey struct {
		PkgPath      string
		Name         string
		EventPkgPath string
		EventName    string
	}
	type handlerValue struct {
		hasHandle   bool
		hasHandling bool
	}

	acc := map[handlerKey]handlerValue{}

	appRoot := filepath.Join(repoRoot, "app")
	fset := token.NewFileSet()
	middlewareImportPath := modulePath + "/pkg/middleware"

	err := filepath.WalkDir(appRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if !strings.Contains(filepath.ToSlash(path), "/handlers/") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		dir := filepath.Dir(path)
		relDir, err := filepath.Rel(repoRoot, dir)
		if err != nil {
			return err
		}
		handlerPkgPath := modulePath + "/" + filepath.ToSlash(relDir)

		imports := map[string]string{}
		for _, imp := range file.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return fmt.Errorf("unquote import %s: %w", imp.Path.Value, err)
			}
			name := ""
			if imp.Name != nil {
				name = imp.Name.Name
			} else {
				parts := strings.Split(p, "/")
				name = parts[len(parts)-1]
			}
			imports[name] = p
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name == nil {
				continue
			}
			switch fn.Name.Name {
			case "Handle", "Handling":
			default:
				continue
			}

			recv := receiverIdent(fn.Recv)
			if recv == "" {
				continue
			}

			if fn.Type == nil || fn.Type.Params == nil || len(fn.Type.Params.List) < 2 {
				continue
			}

			// Require *middleware.Context as the first parameter.
			if !isMiddlewareContext(fn.Type.Params.List[0].Type, imports, middlewareImportPath) {
				continue
			}

			eventPkg, eventName, ok := selectorType(fn.Type.Params.List[1].Type)
			if !ok {
				continue
			}

			eventPkgPath, ok := imports[eventPkg]
			if !ok {
				continue
			}

			key := handlerKey{
				PkgPath:      handlerPkgPath,
				Name:         recv,
				EventPkgPath: eventPkgPath,
				EventName:    eventName,
			}
			v := acc[key]
			if fn.Name.Name == "Handle" {
				v.hasHandle = true
			}
			if fn.Name.Name == "Handling" {
				v.hasHandling = true
			}
			acc[key] = v
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	out := make([]handlerType, 0, len(acc))
	for k, v := range acc {
		if !v.hasHandle {
			continue
		}
		out = append(out, handlerType{
			PkgPath:      k.PkgPath,
			Name:         k.Name,
			EventPkgPath: k.EventPkgPath,
			EventName:    k.EventName,
			HasHandling:  v.hasHandling,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].PkgPath != out[j].PkgPath {
			return out[i].PkgPath < out[j].PkgPath
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		if out[i].EventPkgPath != out[j].EventPkgPath {
			return out[i].EventPkgPath < out[j].EventPkgPath
		}
		return out[i].EventName < out[j].EventName
	})
	return out, nil
}

func receiverIdent(fl *ast.FieldList) string {
	if fl == nil || len(fl.List) != 1 {
		return ""
	}
	t := fl.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if ident, ok := t.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

func isMiddlewareContext(expr ast.Expr, imports map[string]string, middlewareImportPath string) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	pkg, name, ok := selectorType(star.X)
	if !ok || name != "Context" {
		return false
	}
	return imports[pkg] == middlewareImportPath
}

func selectorType(expr ast.Expr) (pkg string, name string, ok bool) {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", "", false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", "", false
	}
	return pkgIdent.Name, sel.Sel.Name, true
}

func defaultAlias(modulePath, importPath string) string {
	if importPath == modulePath+"/pkg/middleware" {
		return "middleware"
	}

	rel := strings.TrimPrefix(importPath, modulePath+"/")
	rel = strings.TrimPrefix(rel, "app/domains/")
	parts := strings.Split(rel, "/")
	if len(parts) >= 2 {
		return parts[0] + "_" + parts[len(parts)-1]
	}
	if len(parts) == 1 && parts[0] != "" {
		return parts[0]
	}
	return ""
}
