package main

import (
	"strings"
	"testing"

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
	if err := parsed.UnmarshalCedar([]byte(src)); err != nil {
		t.Fatal(err)
	}
	if _, err := parsed.Resolve(); err != nil {
		t.Fatal(err)
	}

	got, err := renderModuleModels(parsed.AST(), "drinks")
	if err != nil {
		t.Fatal(err)
	}
	normalized := strings.Join(strings.Fields(string(got)), " ")
	for _, want := range []string{
		`DrinkType cedar.EntityType = "Mixology::Drink"`,
		`ActionAddIce = cedar.NewEntityUID(ActionType, "add_ice")`,
		`Name string`,
		`Featured bool`,
		`Owner cedar.EntityUID`,
		`DrinkNameAttr: cedar.String(m.Name)`,
	} {
		if !strings.Contains(normalized, want) {
			t.Errorf("generated source missing %q:\n%s", want, got)
		}
	}
}
