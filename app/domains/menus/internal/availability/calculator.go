package availability

import (
	"sort"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	drinksq "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	ingredientsq "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
	inventoryq "github.com/TheFellow/go-modular-monolith/app/domains/inventory/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
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

func (c *AvailabilityCalculator) Calculate(ctx *middleware.Context, drinkID entity.DrinkID) models.Availability {
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
	IngredientID  entity.IngredientID
	Required      measurement.Amount
	Available     measurement.Amount
	HasSubstitute bool
}

type AppliedSubstitution struct {
	Original      entity.IngredientID
	Substitute    entity.IngredientID
	Ratio         float64
	QualityImpact ingredientsmodels.Quality
}

func (c *AvailabilityCalculator) CalculateDetail(ctx *middleware.Context, drinkID entity.DrinkID) (Detail, error) {
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
				Available:     measurement.MustAmount(0, req.Amount.Unit()),
				HasSubstitute: hasSub,
			})
			continue
		}

		threshold := pick.Required.Mul(3)
		if pick.Available.Value() < threshold.Value() {
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
	IngredientID     entity.IngredientID
	Required         measurement.Amount
	Available        measurement.Amount
	UsedSubstitution bool
	Ratio            float64
	QualityImpact    ingredientsmodels.Quality
}

type candidate struct {
	id            entity.IngredientID
	required      measurement.Amount
	isOriginal    bool
	ratio         float64
	qualityImpact ingredientsmodels.Quality
}

func (c *AvailabilityCalculator) PickIngredient(ctx store.Context, req drinksmodels.RecipeIngredient) (PickResult, bool) {
	candidates := make([]candidate, 0, 1+len(req.Substitutes))
	candidates = append(candidates, candidate{
		id:            req.IngredientID,
		required:      req.Amount,
		isOriginal:    true,
		ratio:         1,
		qualityImpact: ingredientsmodels.QualityEquivalent,
	})

	seen := map[string]struct{}{req.IngredientID.String(): {}}

	addCandidate := func(id entity.IngredientID, ratio float64, quality ingredientsmodels.Quality) {
		key := id.String()
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, candidate{
			id:            id,
			required:      req.Amount.Mul(ratio),
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
				return rules[i].SubstituteID.String() < rules[j].SubstituteID.String()
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
		available, err := stock.Amount.Convert(cand.required.Unit())
		if err != nil {
			continue
		}
		if available.Value() < cand.required.Value() {
			continue
		}
		picks = append(picks, PickResult{
			IngredientID:     cand.id,
			Required:         cand.required,
			Available:        available,
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
		if a.Available.Value() != b.Available.Value() {
			return a.Available.Value() > b.Available.Value()
		}
		return a.IngredientID.String() < b.IngredientID.String()
	})

	best := picks[0]
	if best.Required.Value() <= 0 {
		return PickResult{}, false
	}

	return best, true
}
