package availability

import (
	"context"
	"sort"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/drinks/models"
	drinksq "github.com/TheFellow/go-modular-monolith/app/drinks/queries"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	ingredientsq "github.com/TheFellow/go-modular-monolith/app/ingredients/queries"
	inventoryq "github.com/TheFellow/go-modular-monolith/app/inventory/queries"
	"github.com/TheFellow/go-modular-monolith/app/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type AvailabilityCalculator struct {
	drinks      *drinksq.Queries
	inventory   *inventoryq.Queries
	ingredients *ingredientsq.Queries
}

func New() *AvailabilityCalculator {
	return &AvailabilityCalculator{
		drinks:      drinksq.New(),
		inventory:   inventoryq.New(),
		ingredients: ingredientsq.New(),
	}
}

func (c *AvailabilityCalculator) Calculate(ctx *middleware.Context, drinkID cedar.EntityUID) models.Availability {
	detail, err := c.CalculateDetail(ctx, drinkID)
	if err != nil {
		return models.AvailabilityUnavailable
	}
	return detail.Status
}

type Detail struct {
	Status        models.Availability
	Missing       []MissingIngredient
	Substitutions []AppliedSubstitution
}

type MissingIngredient struct {
	IngredientID  cedar.EntityUID
	Required      float64
	Available     float64
	HasSubstitute bool
}

type AppliedSubstitution struct {
	Original      cedar.EntityUID
	Substitute    cedar.EntityUID
	Ratio         float64
	QualityImpact ingredientsmodels.Quality
}

func (c *AvailabilityCalculator) CalculateDetail(ctx *middleware.Context, drinkID cedar.EntityUID) (Detail, error) {
	drink, err := c.drinks.Get(ctx, drinkID)
	if err != nil {
		return Detail{}, err
	}

	limited := false
	var missing []MissingIngredient
	var substitutions []AppliedSubstitution

	for _, req := range drink.Recipe.Ingredients {
		if req.Optional {
			continue
		}

		pick, ok := c.PickIngredient(ctx, req)
		if !ok {
			hasSub := len(req.Substitutes) > 0
			if !hasSub && c.ingredients != nil {
				if rules, err := c.ingredients.SubstitutionsFor(ctx, req.IngredientID); err == nil {
					hasSub = len(rules) > 0
				}
			}
			missing = append(missing, MissingIngredient{
				IngredientID:  req.IngredientID,
				Required:      req.Amount,
				Available:     0,
				HasSubstitute: hasSub,
			})
			continue
		}

		if pick.AvailableQty < pick.RequiredQty*3 {
			limited = true
		}
		if pick.UsedSubstitution {
			substitutions = append(substitutions, AppliedSubstitution{
				Original:      req.IngredientID,
				Substitute:    pick.IngredientID,
				Ratio:         pick.Ratio,
				QualityImpact: pick.QualityImpact,
			})
		}
	}

	if len(missing) > 0 {
		return Detail{Status: models.AvailabilityUnavailable, Missing: missing, Substitutions: substitutions}, nil
	}
	if limited {
		return Detail{Status: models.AvailabilityLimited, Missing: nil, Substitutions: substitutions}, nil
	}
	return Detail{Status: models.AvailabilityAvailable, Missing: nil, Substitutions: substitutions}, nil
}

type PickResult struct {
	IngredientID     cedar.EntityUID
	RequiredQty      float64
	AvailableQty     float64
	UsedSubstitution bool
	Ratio            float64
	QualityImpact    ingredientsmodels.Quality
}

type candidate struct {
	id            cedar.EntityUID
	requiredQty   float64
	isOriginal    bool
	ratio         float64
	qualityImpact ingredientsmodels.Quality
}

func (c *AvailabilityCalculator) PickIngredient(ctx context.Context, req drinksmodels.RecipeIngredient) (PickResult, bool) {
	candidates := make([]candidate, 0, 1+len(req.Substitutes))
	candidates = append(candidates, candidate{
		id:            req.IngredientID,
		requiredQty:   req.Amount,
		isOriginal:    true,
		ratio:         1,
		qualityImpact: ingredientsmodels.QualityEquivalent,
	})

	seen := map[string]struct{}{string(req.IngredientID.ID): {}}

	addCandidate := func(id cedar.EntityUID, ratio float64, quality ingredientsmodels.Quality) {
		key := string(id.ID)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, candidate{
			id:            id,
			requiredQty:   req.Amount * ratio,
			isOriginal:    false,
			ratio:         ratio,
			qualityImpact: quality,
		})
	}

	for _, sub := range req.Substitutes {
		if rule, ok := ingredientsmodels.LookupSubstitution(req.IngredientID, sub); ok {
			addCandidate(rule.SubstituteID, rule.Ratio, rule.QualityImpact)
			continue
		}
		addCandidate(sub, 1.0, ingredientsmodels.QualitySimilar)
	}

	if c.ingredients != nil {
		rules, err := c.ingredients.SubstitutionsFor(ctx, req.IngredientID)
		if err == nil {
			sort.Slice(rules, func(i, j int) bool {
				if rules[i].QualityImpact.Rank() != rules[j].QualityImpact.Rank() {
					return rules[i].QualityImpact.Rank() > rules[j].QualityImpact.Rank()
				}
				return string(rules[i].SubstituteID.ID) < string(rules[j].SubstituteID.ID)
			})
			for _, rule := range rules {
				addCandidate(rule.SubstituteID, rule.Ratio, rule.QualityImpact)
			}
		}
	}

	var picks []PickResult
	for _, cand := range candidates {
		stock, err := c.inventory.Get(ctx, cand.id)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			continue
		}
		if stock.Quantity < cand.requiredQty {
			continue
		}
		picks = append(picks, PickResult{
			IngredientID:     cand.id,
			RequiredQty:      cand.requiredQty,
			AvailableQty:     stock.Quantity,
			UsedSubstitution: !cand.isOriginal,
			Ratio:            cand.ratio,
			QualityImpact:    cand.qualityImpact,
		})
	}
	if len(picks) == 0 {
		return PickResult{}, false
	}

	sort.Slice(picks, func(i, j int) bool {
		a := picks[i]
		b := picks[j]

		if !a.UsedSubstitution && b.UsedSubstitution {
			return true
		}
		if a.UsedSubstitution && !b.UsedSubstitution {
			return false
		}
		if a.QualityImpact.Rank() != b.QualityImpact.Rank() {
			return a.QualityImpact.Rank() > b.QualityImpact.Rank()
		}
		if a.AvailableQty != b.AvailableQty {
			return a.AvailableQty > b.AvailableQty
		}
		return string(a.IngredientID.ID) < string(b.IngredientID.ID)
	})

	best := picks[0]
	if best.RequiredQty <= 0 {
		return PickResult{}, false
	}

	return best, true
}
