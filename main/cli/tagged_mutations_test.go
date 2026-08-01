package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	pkglog "github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/urfave/cli/v3"
)

func TestDomainMutationCommandsShareTagsFlag(t *testing.T) {
	t.Parallel()
	c, err := NewCLI()
	testutil.Ok(t, err)
	root := c.Command()

	want := map[string][]string{
		"drinks":      {"create", "update"},
		"ingredients": {"create", "update"},
		"inventory":   {"adjust", "set"},
		"menus":       {"create", "update", "add-drink", "remove-drink", "publish", "draft"},
		"orders":      {"place", "complete", "cancel"},
	}
	seen := make(map[cli.Flag]string)
	for noun, mutations := range want {
		domain := root.Command(noun)
		testutil.NotNil(t, domain)
		for _, mutation := range mutations {
			command := domain.Command(mutation)
			testutil.NotNil(t, command)
			var flag cli.Flag
			for _, candidate := range command.Flags {
				if slices.Contains(candidate.Names(), "tags") {
					flag = candidate
					break
				}
			}
			testutil.NotNil(t, flag)
			if previous, exists := seen[flag]; exists {
				testutil.ErrorIf(t, true, "%s %s reuses mutable tags flag from %s", noun, mutation, previous)
			}
			seen[flag] = noun + " " + mutation
		}
	}
	testutil.Equals(t, len(seen), 15)
}

//nolint:paralleltest // each invocation constructs urfave commands whose flags retain parse state.
func TestMutationTagsReplaceRoundTripAndFilter(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "mutation-tags.db")
	targets := seedCLITagTargets(t, dbPath)

	canonicalInput := `region=east, env=dev, terraform=,"note=coastal,bar"`
	out, err := runTagsCLI(dbPath, "owner", "ingredients", "update",
		"--id", targets.ingredientID, "--name", "CLI Gin Tagged", "--category", "spirit", "--unit", "oz",
		"-tags="+canonicalInput, "--json")
	testutil.Ok(t, err)
	testutil.StringContains(t, out, `"Tags": [`) // The returned mutation state includes the replacement.
	testutil.StringContains(t, out, `"Value": "coastal,bar"`)

	out, err = runTagsCLI(dbPath, "owner", "tags", "list", targets.ingredient)
	testutil.Ok(t, err)
	testutil.Equals(t, out, targets.ingredient+`: env=dev,"note=coastal,bar",region=east,terraform`+"\n")

	// Omitting --tags preserves the complete set.
	_, err = runTagsCLI(dbPath, "owner", "ingredients", "update",
		"--id", targets.ingredientID, "--name", "CLI Gin Preserved", "--category", "spirit", "--unit", "oz")
	testutil.Ok(t, err)
	out, err = runTagsCLI(dbPath, "owner", "tags", "list", targets.ingredient)
	testutil.Ok(t, err)
	testutil.StringContains(t, out, `env=dev,"note=coastal,bar",region=east,terraform`)

	// A tag written through the domain tree is immediately available to the
	// existing cross-domain filter surface.
	out, err = runTagsCLI(dbPath, "owner", "ingredients", "list", "--filter", `tags contains "note=coastal,bar"`, "--json")
	testutil.Ok(t, err)
	testutil.StringContains(t, out, targets.ingredientID)

	// An explicit empty value is distinct from omission and clears the set.
	_, err = runTagsCLI(dbPath, "owner", "ingredients", "update",
		"--id", targets.ingredientID, "--name", "CLI Gin Cleared", "--category", "spirit", "--unit", "oz", "--tags=", "--json")
	testutil.Ok(t, err)
	out, err = runTagsCLI(dbPath, "owner", "tags", "list", targets.ingredient)
	testutil.Ok(t, err)
	testutil.Equals(t, out, targets.ingredient+": (none)\n")

	// Collection parsing precedes the mutation, so invalid input cannot change
	// either the domain entity or its tags.
	_, err = runTagsCLI(dbPath, "owner", "ingredients", "update",
		"--id", targets.ingredientID, "--name", "MUST NOT PERSIST", "--category", "spirit", "--unit", "oz",
		"--tags", "region=east,region=west")
	assertCLIExitCode(t, err, errors.ExitInvalid)
	out, err = runTagsCLI(dbPath, "owner", "ingredients", "get", "--id", targets.ingredientID, "--json")
	testutil.Ok(t, err)
	testutil.StringContains(t, out, "CLI Gin Cleared")
	testutil.ErrorIf(t, strings.Contains(out, "MUST NOT PERSIST"), "invalid tags allowed mutation: %s", out)
}

//nolint:paralleltest // each invocation constructs urfave commands whose flags retain parse state.
func TestDomainMutationTagsFilterAcrossEveryOperationalDomain(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "domain-filter-tags.db")
	targets := seedCLITagTargets(t, dbPath)
	drinkFile := filepath.Join(dir, "drink.json")
	drinkJSON := fmt.Sprintf(`{
  "id": %q,
  "name": "CLI Highball",
  "category": "highball",
  "glass": "highball",
  "recipe": {
    "ingredients": [{"ingredient_id": %q, "amount": 1, "unit": "oz"}],
    "steps": ["build"]
  }
}`, targets.drink, targets.ingredientID)
	testutil.Ok(t, os.WriteFile(drinkFile, []byte(drinkJSON), 0o600))

	mutations := [][]string{
		{"drinks", "update", "--file", drinkFile, "--tags=origin=domain-cli"},
		{"ingredients", "update", "--id", targets.ingredientID, "--name", "CLI Gin", "--category", "spirit", "--unit", "oz", "--tags=origin=domain-cli"},
		{"inventory", "adjust", "--ingredient-id", targets.ingredientID, "--delta", "0", "--reason", "corrected", "--tags=origin=domain-cli"},
		{"menus", "draft", "--id", targets.menu, "--tags=origin=domain-cli"},
		{"orders", "complete", "--id", targets.order, "--tags=origin=domain-cli"},
	}
	for _, args := range mutations {
		_, err := runTagsCLI(dbPath, "owner", args...)
		if err != nil {
			testutil.ErrorIf(t, true, "%v: %v", args[:2], err)
		}
	}

	for _, tc := range []struct {
		noun   string
		target string
	}{
		{"drinks", targets.drink},
		{"ingredients", targets.ingredient},
		{"inventory", targets.inventory},
		{"menus", targets.menu},
		{"orders", targets.order},
	} {
		out, err := runTagsCLI(dbPath, "owner", tc.noun, "list", "--filter", `tags contains "origin=domain-cli"`, "--json")
		testutil.Ok(t, err)
		testutil.StringContains(t, out, tc.target)
	}
}

type rejectedTagTarget struct {
	tags tag.Tags
}

func (e *rejectedTagTarget) EntityUID() cedar.EntityUID {
	return cedar.NewEntityUID("Unsupported", "target")
}

func (e *rejectedTagTarget) SetTags(tags tag.Tags) { e.tags = tags }

func TestTaggedMutationRollsBackWhenTagReplacementFails(t *testing.T) {
	t.Parallel()
	parent := authn.ToContext(context.Background(), authn.Owner())
	parent = pkglog.ToContext(parent, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s, err := store.Open(parent, filepath.Join(t.TempDir(), "rollback.db"))
	testutil.Ok(t, err)
	a := app.New(parent, app.Config{Store: s})
	t.Cleanup(func() { testutil.Ok(t, a.Close()) })
	ctx := middleware.NewContext(parent)

	created, err := a.Ingredients.Create(ctx, &models.Ingredient{
		Name: "Before", Category: models.CategorySpirit, Unit: "oz",
	})
	testutil.Ok(t, err)
	auditBefore, err := a.Audit.Count(ctx, audit.ListRequest{})
	testutil.Ok(t, err)

	c := &CLI{app: a}
	command := &cli.Command{Flags: []cli.Flag{tagsFlag()}}
	testutil.Ok(t, command.Set("tags", "region=east"))
	_, err = runTaggedMutation(c, ctx, command, func(txCtx *middleware.Context) (*rejectedTagTarget, error) {
		_, updateErr := a.Ingredients.Update(txCtx, &models.Ingredient{
			ID: created.ID, Name: "After", Category: models.CategorySpirit, Unit: "oz",
		})
		return &rejectedTagTarget{}, updateErr
	})
	testutil.ErrorIf(t, err == nil, "expected tag replacement to fail")

	got, err := a.Ingredients.Get(ctx, created.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got.Name, "Before")
	auditAfter, err := a.Audit.Count(ctx, audit.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, auditAfter, auditBefore)
}
