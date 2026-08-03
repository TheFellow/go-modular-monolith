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
		`ResourceType cedar.EntityType = DrinkType`,
		`ActionAddIce = cedar.NewEntityUID(ActionType, "add_ice")`,
		`Name string`,
		`Featured bool`,
		`Owner cedar.EntityUID`,
		`DrinkNameAttr: cedar.String(m.Name)`,
		`DrinkOwnerAttr: cedar.NewEntityUID("Mixology::Actor", m.Owner.ID)`,
		`func ValidateEntity(entity cedar.Entity) error`,
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
		`Owner: cedar.NewEntityUID("Mixology::Actor", "test-owner")`,
		`moduleauthz.DrinkNameAttr: cedar.String("test-name")`,
		`testutil.Equals(t, got, want)`,
		`validate.New(resolved).Entity(got)`,
		`moduleauthz.ValidateEntity(got)`,
	} {
		testutil.ErrorIf(t, !strings.Contains(testSource, want), "generated test source missing %q:\n%s", want, generatedTests)
	}

	for _, source := range [][]byte{got, generatedTests} {
		testutil.ErrorIf(t, strings.Contains(string(source), "app/kernel/entity"),
			"generated authz code depends on the kernel entity generator:\n%s", source)
	}
}

func TestRenderModuleModelsWithStringTags(t *testing.T) {
	t.Parallel()

	const src = `
namespace Mixology {
    entity Actor;
    entity Drink { Name: String } tags String;
}
namespace Mixology::Drink {
    action get appliesTo {
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
		`Tags map[string]string`,
		`tags := make(cedar.RecordMap, len(m.Tags))`,
		`tags[cedar.String(key)] = cedar.String(value)`,
		`Tags: cedar.NewRecord(tags)`,
	} {
		testutil.ErrorIf(t, !strings.Contains(normalized, want), "generated source missing %q:\n%s", want, got)
	}

	generatedTests, err := renderModuleModelTests(parsed.AST(), "drinks",
		"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz")
	testutil.Ok(t, err)
	testSource := strings.Join(strings.Fields(string(generatedTests)), " ")
	for _, want := range []string{
		`Tags: map[string]string{"audience": "members", "featured": ""}`,
		`"audience": cedar.String("members")`,
		`"featured": cedar.String("")`,
	} {
		testutil.ErrorIf(t, !strings.Contains(testSource, want), "generated test source missing %q:\n%s", want, generatedTests)
	}
}

func TestRenderModuleModelsRejectsUnsupportedTagTypes(t *testing.T) {
	t.Parallel()

	const src = `
namespace Mixology {
    entity Actor;
    entity Drink tags Long;
}
namespace Mixology::Drink {
    action get appliesTo {
        principal: Mixology::Actor,
        resource: Mixology::Drink,
        context: {}
    };
}`

	var parsed schema.Schema
	testutil.Ok(t, parsed.UnmarshalCedar([]byte(src)))
	_, err := parsed.Resolve()
	testutil.Ok(t, err)

	_, err = renderModuleModels(parsed.AST(), "drinks")
	testutil.ErrorIf(t, err == nil, "expected unsupported tag type error")
	testutil.ErrorIf(t, !strings.Contains(err.Error(), "Long"), "error does not identify unsupported type: %v", err)
	testutil.ErrorIf(t, !strings.Contains(err.Error(), "only String is supported"), "unexpected error: %v", err)

	_, err = renderModuleModelTests(parsed.AST(), "drinks", "example.com/drinks/authz")
	testutil.ErrorIf(t, err == nil, "expected unsupported tag type error")
	testutil.ErrorIf(t, !strings.Contains(err.Error(), "only String is supported"), "unexpected error: %v", err)
}

func TestRenderModuleModelsResolvesCommonTypeAliases(t *testing.T) {
	t.Parallel()

	const src = `
namespace Mixology {
    type Label = String;
    entity Actor;
    entity Drink { Name: Label };
}
namespace Mixology::Drink {
    action get appliesTo {
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
	testutil.ErrorIf(t, !strings.Contains(normalized, `Name string`),
		"common String alias generated the wrong field type:\n%s", got)
}

func TestRenderModuleModelsRejectsUnsupportedCommonTypes(t *testing.T) {
	t.Parallel()

	const src = `
namespace Mixology {
    type Details = { note: String };
    entity Actor;
    entity Drink { Details: Details };
}
namespace Mixology::Drink {
    action get appliesTo {
        principal: Mixology::Actor,
        resource: Mixology::Drink,
        context: {}
    };
}`

	var parsed schema.Schema
	testutil.Ok(t, parsed.UnmarshalCedar([]byte(src)))
	_, err := parsed.Resolve()
	testutil.Ok(t, err)

	_, err = renderModuleModels(parsed.AST(), "drinks")
	testutil.ErrorIf(t, err == nil, "expected unsupported common type error")
	testutil.ErrorIf(t, !strings.Contains(err.Error(), "unsupported Cedar type"), "unexpected error: %v", err)
}

func TestRenderModuleModelsRejectsParentTypes(t *testing.T) {
	t.Parallel()

	const src = `
namespace Mixology {
    entity Actor;
    entity Catalog;
    entity Drink in [Catalog];
}
namespace Mixology::Drink {
    action get appliesTo {
        principal: Mixology::Actor,
        resource: Mixology::Drink,
        context: {}
    };
}`

	var parsed schema.Schema
	testutil.Ok(t, parsed.UnmarshalCedar([]byte(src)))
	_, err := parsed.Resolve()
	testutil.Ok(t, err)

	_, err = renderModuleModels(parsed.AST(), "drinks")
	testutil.ErrorIf(t, err == nil, "expected unsupported parent type error")
	testutil.ErrorIf(t, !strings.Contains(err.Error(), "parent types are not supported"), "unexpected error: %v", err)
}

func TestRenderModuleModelsRejectsActionContexts(t *testing.T) {
	t.Parallel()

	const src = `
namespace Mixology {
    entity Actor;
    entity Drink;
}
namespace Mixology::Drink {
    action get appliesTo {
        principal: Mixology::Actor,
        resource: Mixology::Drink,
        context: { source: String }
    };
}`

	var parsed schema.Schema
	testutil.Ok(t, parsed.UnmarshalCedar([]byte(src)))
	_, err := parsed.Resolve()
	testutil.Ok(t, err)

	_, err = renderModuleModels(parsed.AST(), "drinks")
	testutil.ErrorIf(t, err == nil, "expected unsupported action context error")
	testutil.ErrorIf(t, !strings.Contains(err.Error(), "non-empty action contexts are not supported"), "unexpected error: %v", err)
}

func TestRenderModuleModelsRejectsReservedAttributeNames(t *testing.T) {
	t.Parallel()

	for _, attr := range []string{`"uID"`, `tags`, `cedarEntity`} {
		t.Run(attr, func(t *testing.T) {
			t.Parallel()
			src := `
namespace Mixology {
    entity Actor;
    entity Drink { ` + attr + `: String };
}
namespace Mixology::Drink {
    action get appliesTo {
        principal: Mixology::Actor,
        resource: Mixology::Drink,
        context: {}
    };
}`

			var parsed schema.Schema
			testutil.Ok(t, parsed.UnmarshalCedar([]byte(src)))
			_, err := parsed.Resolve()
			testutil.Ok(t, err)

			_, err = renderModuleModels(parsed.AST(), "drinks")
			testutil.ErrorIf(t, err == nil, "expected reserved attribute name error")
			testutil.ErrorIf(t, !strings.Contains(err.Error(), "conflicts with the generated"), "unexpected error: %v", err)
		})
	}
}

func TestRenderModuleModelsRejectsActionNameCollisions(t *testing.T) {
	t.Parallel()

	const src = `
namespace Mixology {
    entity Actor;
    entity Drink;
}
namespace Mixology::Drink {
    action "add-ice", add_ice appliesTo {
        principal: Mixology::Actor,
        resource: Mixology::Drink,
        context: {}
    };
}`

	var parsed schema.Schema
	testutil.Ok(t, parsed.UnmarshalCedar([]byte(src)))
	_, err := parsed.Resolve()
	testutil.Ok(t, err)

	_, err = renderModuleModels(parsed.AST(), "drinks")
	testutil.ErrorIf(t, err == nil, "expected normalized action name collision")
	testutil.ErrorIf(t, !strings.Contains(err.Error(), `both normalize to variable name "ActionAddIce"`), "unexpected error: %v", err)
}

func TestRenderModuleModelsRejectsNormalizedNameCollisions(t *testing.T) {
	t.Parallel()

	const src = `
namespace Mixology {
    entity Actor;
    entity Drink { "foo-bar": String, foo_bar: Long };
}
namespace Mixology::Drink {
    action "add-ice", add_ice appliesTo {
        principal: Mixology::Actor,
        resource: Mixology::Drink,
        context: {}
    };
}`

	var parsed schema.Schema
	testutil.Ok(t, parsed.UnmarshalCedar([]byte(src)))
	_, err := parsed.Resolve()
	testutil.Ok(t, err)

	_, err = renderModuleModels(parsed.AST(), "drinks")
	testutil.ErrorIf(t, err == nil, "expected normalized name collision")
	testutil.ErrorIf(t, !strings.Contains(err.Error(), `both normalize to field name "FooBar"`), "unexpected error: %v", err)
}
