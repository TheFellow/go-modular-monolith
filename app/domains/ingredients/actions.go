package ingredients

import (
	"context"

	ingredientauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	pkgAuthz "github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	cedar "github.com/cedar-policy/cedar-go"
)

// Stable control identities let every presentation adapter bind its native
// controls to the same ingredient capabilities.
const (
	ControlList   actions.ID = "ingredients.list"
	ControlCreate actions.ID = "ingredients.create"
	ControlEdit   actions.ID = "ingredients.edit"
	ControlDelete actions.ID = "ingredients.delete"
	ControlTags   actions.ID = "ingredients.tags"
)

// ActionProjector produces framework-neutral ingredient control state.
// Transient concerns such as dirty forms and in-flight requests stay local to
// each presentation adapter.
type ActionProjector struct {
	Authorize pkgAuthz.EntityAuthorizer
}

func NewActionProjector() ActionProjector {
	return ActionProjector{Authorize: pkgAuthz.AuthorizeEntity}
}

// Project returns create state and, when selected is non-nil, all actions on
// that ingredient. Permission denials hide controls; evaluator failures are
// returned to the presentation adapter.
func (p ActionProjector) Project(ctx context.Context, principal cedar.EntityUID, selected *models.Ingredient) ([]actions.State, error) {
	authorize := p.Authorize
	if authorize == nil {
		authorize = NewActionProjector().Authorize
	}
	permission := func(action cedar.EntityUID, resource cedar.Entity) actions.Permission {
		return actions.Require(func(ctx context.Context) error {
			return authorize(ctx, principal, action, resource)
		})
	}

	collection := models.Ingredient{ID: entity.IngredientID(cedar.NewEntityUID(entity.TypeIngredient, "workspace"))}.CedarEntity()
	declaration := actions.Group{Controls: []actions.Control{
		{ID: ControlList, Permission: permission(ingredientauthz.ActionList, collection)},
		{ID: ControlCreate, Permission: permission(ingredientauthz.ActionCreate, (models.Ingredient{}).CedarEntity())},
	}}
	if selected == nil {
		return actions.Evaluate(ctx, declaration)
	}

	resource := selected.CedarEntity()
	declaration.Controls = append(declaration.Controls,
		actions.Control{ID: ControlEdit, Permission: permission(ingredientauthz.ActionUpdate, resource)},
		actions.Control{ID: ControlDelete, Permission: permission(ingredientauthz.ActionDelete, resource)},
		actions.Control{ID: ControlTags, Permission: permission(ingredientauthz.ActionTag, resource)},
	)
	return actions.Evaluate(ctx, declaration)
}
