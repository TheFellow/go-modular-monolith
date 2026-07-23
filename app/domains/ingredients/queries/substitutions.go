package queries

import (
	"strings"

	ingredientsdao "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) SubstitutionsFor(ctx store.Context, ingredientID entity.IngredientID) ([]models.SubstitutionRule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Keep direct catalog-ID lookup useful for callers that work with the
	// symbolic IDs declared by the catalog itself.
	if rules := models.SubstitutionsFor(ingredientID); len(rules) > 0 {
		return rules, nil
	}

	// Persisted ingredients use generated IDs. Resolve the catalog's stable
	// name keys to those live IDs so catalog substitutions participate in the
	// same public query contract as recipe-declared substitutes.
	original, err := q.Get(ctx, ingredientID)
	if err != nil {
		return nil, err
	}
	originalKey := substitutionKey(original.Name)

	byKey := make(map[string]entity.IngredientID)
	for ingredient, err := range q.dao.List(ctx, ingredientsdao.ListFilter{}) {
		if err != nil {
			return nil, err
		}
		byKey[substitutionKey(ingredient.Name)] = ingredient.ID
	}

	var resolved []models.SubstitutionRule
	for _, rule := range models.DefaultSubstitutionRules() {
		if rule.IngredientID.String() != originalKey {
			continue
		}
		substituteID, ok := byKey[rule.SubstituteID.String()]
		if !ok {
			continue
		}
		rule.IngredientID = original.ID
		rule.SubstituteID = substituteID
		resolved = append(resolved, rule)
	}
	return resolved, nil
}

func substitutionKey(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(name)), "-"))
}
