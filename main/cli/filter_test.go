package main

import (
	"bytes"
	"context"
	stderrors "errors"
	"strings"
	"testing"

	auditmodels "github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menusmodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/filter"
	"github.com/urfave/cli/v3"
)

func TestFilterHelpUsesConcreteSchema(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := writeFilterHelp(&out, models.ListFilterSchema()); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{
		"category", "Ingredient category", "&& / and", "value.contains", `--filter 'category == "spirit"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("help does not contain %q:\n%s", want, text)
		}
	}
	for _, example := range models.ListFilterSchema().Examples() {
		if _, err := filter.Parse(models.ListFilterSchema(), example); err != nil {
			t.Fatalf("generated example %q does not parse: %v", example, err)
		}
	}
}

func TestEveryGeneratedFilterExampleParses(t *testing.T) {
	t.Parallel()

	checkFilterExamples(t, auditmodels.ListFilterSchema())
	checkFilterExamples(t, drinksmodels.ListFilterSchema())
	checkFilterExamples(t, models.ListFilterSchema())
	checkFilterExamples(t, inventorymodels.ListFilterSchema())
	checkFilterExamples(t, menusmodels.ListFilterSchema())
	checkFilterExamples(t, ordersmodels.ListFilterSchema())
}

func checkFilterExamples[T any](t *testing.T, schema filter.Schema[T]) {
	t.Helper()
	for _, example := range schema.Examples() {
		if _, err := filter.Parse(schema, example); err != nil {
			t.Errorf("generated example %q does not parse: %v", example, err)
		}
	}
}

func TestFilterHelpDoesNotOpenApplicationOrRequireScopeArgument(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		{"mixology", "ingredients", "list", "--filter-help"},
		{"mixology", "audit", "history", "--filter-help"},
		{"mixology", "audit", "actor", "--filter-help"},
	} {
		c, err := NewCLI()
		if err != nil {
			t.Fatal(err)
		}
		c.dbPath = t.TempDir() // opening a directory would fail if Before reached storage
		cmd := c.Command()
		var out bytes.Buffer
		leaf := cmd.Command(args[1]).Command(args[2])
		leaf.Writer = &out
		leaf.ErrWriter = &out
		if err := cmd.Run(context.Background(), args); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if !strings.Contains(out.String(), "FILTER SYNTAX") {
			t.Fatalf("%v output:\n%s", args, out.String())
		}
	}
}

func TestEveryListCommandHasFilterFlags(t *testing.T) {
	t.Parallel()

	c, err := NewCLI()
	if err != nil {
		t.Fatal(err)
	}
	for _, noun := range c.Command().Commands {
		for _, command := range noun.Commands {
			if command.Name != "list" {
				continue
			}
			names := map[string]bool{}
			for _, flag := range command.Flags {
				names[flag.Names()[0]] = true
			}
			if !names["filter"] || !names["filter-help"] {
				t.Errorf("%s list filter flags = %v", noun.Name, names)
			}
		}
	}
}

func TestAuditScopeArgumentRemainsRequiredWithoutFilterHelp(t *testing.T) {
	t.Parallel()

	for _, scope := range []string{"history", "actor"} {
		c, err := NewCLI()
		if err != nil {
			t.Fatal(err)
		}
		c.dbPath = t.TempDir() + "/test.db"
		cmd := c.Command()
		var out bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &out, &out
		err = cmd.Run(context.Background(), []string{"mixology", "audit", scope})
		var exit cli.ExitCoder
		if !stderrors.As(err, &exit) || exit.ExitCode() != 2 {
			t.Fatalf("audit %s error = %v, want usage exit", scope, err)
		}
	}
}
