package main

import (
	_ "embed"
	"os"
	"strings"
	"text/template"

	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
)

//go:embed entities.go.tpl
var tmplText string

func main() {
	f, err := os.Create("entities_gen.go")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	tmpl := template.Must(template.New("entity").Funcs(template.FuncMap{
		"lower": strings.ToLower,
	}).Parse(tmplText))

	if err := tmpl.Execute(f, entity.Entities); err != nil {
		panic(err)
	}
}
