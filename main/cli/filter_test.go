package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	auditmodels "github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menusmodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/filter"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/urfave/cli/v3"
)

func TestFilterHelpUsesConcreteSchema(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	testutil.Ok(t, writeFilterHelp(&out, models.ListFilterSchema()))
	text := out.String()
	for _, want := range []string{
		"category", "Ingredient category", "&& / and", "value.contains", `--filter 'category == "spirit"`,
	} {
		testutil.ErrorIf(t, !strings.Contains(text, want), "help does not contain %q:\n%s", want, text)
	}
	for _, example := range models.ListFilterSchema().Examples() {
		_, err := filter.Parse(models.ListFilterSchema(), example)
		testutil.ErrorIf(t, err != nil, "generated example %q does not parse: %v", example, err)
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
		_, err := filter.Parse(schema, example)
		testutil.ErrorIf(t, err != nil, "generated example %q does not parse: %v", example, err)
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
		testutil.Ok(t, err)
		c.dbPath = t.TempDir() // opening a directory would fail if Before reached storage
		cmd := c.Command()
		var out bytes.Buffer
		leaf := cmd.Command(args[1]).Command(args[2])
		leaf.Writer = &out
		leaf.ErrWriter = &out
		testutil.Ok(t, cmd.Run(context.Background(), args))
		testutil.ErrorIf(t, !strings.Contains(out.String(), "FILTER SYNTAX"), "%v output:\n%s", args, out.String())
	}
}

func TestEveryListCommandHasFilterFlags(t *testing.T) {
	t.Parallel()

	c, err := NewCLI()
	testutil.Ok(t, err)
	for _, noun := range c.Command().Commands {
		for _, command := range noun.Commands {
			if command.Name != "list" {
				continue
			}
			names := map[string]bool{}
			for _, flag := range command.Flags {
				names[flag.Names()[0]] = true
			}
			testutil.ErrorIf(t, !names["filter"] || !names["filter-help"], "%s list filter flags = %v", noun.Name, names)
		}
	}
}

func TestAuditScopeArgumentRemainsRequiredWithoutFilterHelp(t *testing.T) {
	t.Parallel()

	for _, scope := range []string{"history", "actor"} {
		c, err := NewCLI()
		testutil.Ok(t, err)
		c.dbPath = t.TempDir() + "/test.db"
		cmd := c.Command()
		var out bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &out, &out
		err = cmd.Run(context.Background(), []string{"mixology", "audit", scope})
		var exit cli.ExitCoder
		testutil.ErrorAs(t, err, &exit)
		testutil.Equals(t, exit.ExitCode(), 2)
	}
}
