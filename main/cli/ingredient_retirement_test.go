//nolint:paralleltest // CLI integration owns a persistent database lifecycle.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestIngredientsCLIRetireWithExplicitReplacement(t *testing.T) {
	dir := t.TempDir()
	cli := newCLIE2E(filepath.Join(dir, "retirement.db"))
	retired := strings.TrimSpace(cli.Run("ingredients", "create", "Herradura", "--category", "spirit", "--unit", "oz").Stdout)
	replacement := strings.TrimSpace(cli.Run("ingredients", "create", "Hornitos", "--category", "spirit", "--unit", "oz").Stdout)
	input := filepath.Join(dir, "drink.json")
	testutil.Ok(t, os.WriteFile(input, []byte(`{"name":"House Margarita","category":"cocktail","glass":"coupe","recipe":{"ingredients":[{"ingredient_id":"`+retired+`","amount":1,"unit":"oz"}],"steps":["shake"]}}`), 0o600))
	drink := cli.Run("drinks", "create", "--file", input)
	testutil.Ok(t, drink.Err)
	drinkID := strings.TrimSpace(drink.Stdout)
	result := cli.Run("ingredients", "retire", "--id", retired, "--replacement-id", replacement, "--replacement-ratio", "1")
	testutil.Ok(t, result.Err)
	shown := cli.Run("drinks", "get", "--id", drinkID, "--json")
	testutil.Ok(t, shown.Err)
	testutil.StringContains(t, shown.Stdout, replacement)
	testutil.StringContains(t, shown.Stdout, `"status": "active"`)
}
