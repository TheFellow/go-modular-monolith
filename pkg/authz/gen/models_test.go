package main

import (
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/cedar-policy/cedar-go/x/exp/schema"
)

func TestRenderModuleModels(t *testing.T) {
	t.Parallel()

	const src = `
namespace Mixology {
    entity Actor;
    entity Drink { Name: String, Featured: Bool, Owner: Actor };
}
namespace Mixology::Drink {
    action list, add_ice appliesTo {
        principal: Mixology::Actor,
        resource: Mixology::Drink,
        context: {}
    };
}`

	var parsed schema.Schema
	testutil.Ok(t, parsed.UnmarshalCedar([]byte(src)))
	_, err := parsed.Resolve()
	testutil.Ok(t, err)

	got, err := renderModuleModels(parsed.AST(), "drinks")
	testutil.Ok(t, err)
	normalized := strings.Join(strings.Fields(string(got)), " ")
	for _, want := range []string{
		`DrinkType cedar.EntityType = "Mixology::Drink"`,
		`ActionAddIce = cedar.NewEntityUID(ActionType, "add_ice")`,
		`Name string`,
		`Featured bool`,
		`Owner cedar.EntityUID`,
		`DrinkNameAttr: cedar.String(m.Name)`,
	} {
		testutil.ErrorIf(t, !strings.Contains(normalized, want), "generated source missing %q:\n%s", want, got)
	}

	generatedTests, err := renderModuleModelTests(parsed.AST(), "drinks",
		"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz")
	testutil.Ok(t, err)
	testSource := strings.Join(strings.Fields(string(generatedTests)), " ")
	for _, want := range []string{
		`func TestDrinkCedarEntity(t *testing.T)`,
		`UID: cedar.NewEntityUID("Wrong::Type", "test-id")`,
		`UID: cedar.NewEntityUID(moduleauthz.DrinkType, "test-id")`,
		`moduleauthz.DrinkNameAttr: cedar.String("test-name")`,
		`testutil.Equals(t, got, want)`,
	} {
		testutil.ErrorIf(t, !strings.Contains(testSource, want), "generated test source missing %q:\n%s", want, generatedTests)
	}

	for _, source := range [][]byte{got, generatedTests} {
		testutil.ErrorIf(t, strings.Contains(string(source), "app/kernel/entity"),
			"generated authz code depends on the kernel entity generator:\n%s", source)
	}
}
