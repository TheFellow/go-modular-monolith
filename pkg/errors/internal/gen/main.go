package main

import (
	"bytes"
	"flag"
	"go/format"
	"os"
	"path/filepath"
	"text/template"

	perrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func main() {
	var out string
	flag.StringVar(&out, "out", "errors_gen.go", "output file (relative to pkg/errors)")
	flag.Parse()

	wd, err := os.Getwd()
	must(err)

	tmplBytes, err := os.ReadFile(filepath.Join(wd, "internal", "gen", "errors.go.tpl"))
	must(err)

	tmpl, err := template.New("gen").Parse(string(tmplBytes))
	must(err)

	var buf bytes.Buffer
	must(tmpl.Execute(&buf, struct {
		Kinds []perrors.ErrorKind
	}{
		Kinds: perrors.ErrorKinds,
	}))

	formatted, err := format.Source(buf.Bytes())
	must(err)

	must(os.WriteFile(filepath.Join(wd, out), formatted, 0o644))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
