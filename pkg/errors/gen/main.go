package main

import (
	"bytes"
	"embed"
	"flag"
	"go/format"
	"os"
	"path/filepath"
	"text/template"

	perrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
)

//go:embed errors.go.tpl testutil.go.tpl
var templates embed.FS

func main() {
	var out string
	flag.StringVar(&out, "out", "errors_gen.go", "output file (relative to pkg/errors)")
	flag.Parse()

	wd, err := os.Getwd()
	must(err)

	data := struct {
		Kinds []perrors.ErrorKind
	}{
		Kinds: perrors.ErrorKinds,
	}

	generate(wd, "errors.go.tpl", out, data)
	generate(wd, "testutil.go.tpl", filepath.Join("..", "testutil", "errors_gen.go"), data)
}

func generate(wd, tplFile, outFile string, data any) {
	tmplBytes, err := templates.ReadFile(tplFile)
	must(err)

	tmpl, err := template.New("gen").Parse(string(tmplBytes))
	must(err)

	var buf bytes.Buffer
	must(tmpl.Execute(&buf, data))

	formatted, err := format.Source(buf.Bytes())
	must(err)

	must(os.WriteFile(filepath.Join(wd, outFile), formatted, 0o644))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
