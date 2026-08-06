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
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/set"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type AvailabilityCalculator struct {
	drinks      *drinksq.Queries
	inventory   *inventoryq.Queries
	ingredients *ingredientsq.Queries
}

func New(s *store.Store, tags tag.Repository) *AvailabilityCalculator {
	return &AvailabilityCalculator{
		drinks:      drinksq.New(s, tags),
		inventory:   inventoryq.New(s, tags),
		ingredients: ingredientsq.New(s, tags),
	}
}

func (c *AvailabilityCalculator) Calculate(ctx store.Context, drinkID entity.DrinkID) models.Availability {
	// Availability is a user-facing readiness signal, so dependency failures
	// degrade to "unavailable" instead of surfacing infrastructure errors in
	// menu rendering.
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

func (c *AvailabilityCalculator) Readiness(ctx store.Context, menu *models.Menu) (models.ReadinessReport, error) {
	report := models.ReadinessReport{MenuID: menu.ID, Status: menu.Status}
	for _, item := range menu.Items {
		drink, err := c.drinks.Get(ctx, item.DrinkID)
		if err != nil {
			return models.ReadinessReport{}, err
		}
		if drink.Status == drinksmodels.StatusReviewRequired {
			report.Findings = append(report.Findings, models.ReadinessFinding{
				Severity: models.ReadinessBlocker, Code: models.ReadinessReviewRequired, DrinkID: drink.ID,
				Message: "drink " + drink.ID.String() + " requires recipe review",
			})
		}
		detail, err := c.CalculateDetail(ctx, drink.ID)
		if err != nil {
			return models.ReadinessReport{}, err
		}
		for _, missing := range detail.Missing {
			if _, ingredientErr := c.ingredients.Get(ctx, missing.IngredientID); ingredientErr != nil {
				if !errors.IsNotFound(ingredientErr) {
					return models.ReadinessReport{}, ingredientErr
				}
				report.Findings = append(report.Findings, models.ReadinessFinding{
					Severity: models.ReadinessBlocker, Code: models.ReadinessRetiredIngredient, DrinkID: drink.ID, IngredientID: missing.IngredientID,
					Message: "drink " + drink.ID.String() + " references retired or missing ingredient " + missing.IngredientID.String(),
				})
			}
		}
		for _, substitution := range detail.Substitutions {
			report.Findings = append(report.Findings, models.ReadinessFinding{
				Severity: models.ReadinessBlocker, Code: models.ReadinessTemporarySubstitution, DrinkID: drink.ID, IngredientID: substitution.Original,
				Message: "drink " + drink.ID.String() + " relies on temporary substitution " + substitution.Substitute.String() + " for " + substitution.Original.String(),
			})
		}
		switch detail.Status {
		case models.AvailabilityAvailable:
		case models.AvailabilityUnavailable:
			report.Findings = append(report.Findings, models.ReadinessFinding{
				Severity: models.ReadinessBlocker, Code: models.ReadinessUnavailable, DrinkID: drink.ID,
				Message: "drink " + drink.ID.String() + " is unavailable",
			})
		case models.AvailabilityLimited:
			if len(detail.Substitutions) == 0 {
				report.Findings = append(report.Findings, models.ReadinessFinding{
					Severity: models.ReadinessWarning, Code: models.ReadinessLowStock, DrinkID: drink.ID,
					Message: "drink " + drink.ID.String() + " has low stock",
				})
			}
		}
	}
	return report, nil
}

func (c *AvailabilityCalculator) CalculateDetail(ctx store.Context, drinkID entity.DrinkID) (Detail, error) {
	drink, err := c.drinks.Get(ctx, drinkID)
	if err != nil {
		return Detail{}, err
	}
	reviewRequired := drink.Status == drinksmodels.StatusReviewRequired

	limited := false
	var missing []MissingIngredient
	var substitutions []AppliedSubstitution

	requirements := make([]drinksmodels.RecipeIngredient, 0, len(drink.Recipe.Ingredients))
	for _, req := range drink.Recipe.Ingredients {
		if req.Optional {
			continue
		}
		requirements = append(requirements, req)
	}

	picks, fulfilled := c.PickIngredients(ctx, requirements)
	if !fulfilled {
		for _, req := range requirements {
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
		}
		return Detail{Status: models.AvailabilityUnavailable, Missing: missing}, nil
	}

	for i, pick := range picks {
		req := requirements[i]
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

	// A temporary substitute may keep a review-required Drink achievable, but
	// it cannot silently approve a permanent recipe change.
	if limited || reviewRequired || len(substitutions) > 0 {
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
	picks, ok := c.PickIngredients(ctx, []drinksmodels.RecipeIngredient{req})
	if !ok {
		return PickResult{}, false
	}
	return picks[0], true
}

// PickIngredients selects one stock source for each requirement while
// reserving stock selected by earlier choices. Candidate preferences remain
// deterministic, but the search backtracks when a preferred source would
// prevent the complete set of requirements from being fulfilled.
func (c *AvailabilityCalculator) PickIngredients(ctx store.Context, requirements []drinksmodels.RecipeIngredient) ([]PickResult, bool) {
	picks, ok, err := c.PlanIngredients(ctx, requirements)
	if err != nil {
		return nil, false
	}
	return picks, ok
}

// PlanIngredients is the strict fulfillment path used by mutations. Unlike
// menu readiness, it preserves dependency and conversion errors so callers do
// not misreport infrastructure failures as insufficient stock.
func (c *AvailabilityCalculator) PlanIngredients(ctx store.Context, requirements []drinksmodels.RecipeIngredient) ([]PickResult, bool, error) {
	candidateSets := make([][]PickResult, len(requirements))
	for i, req := range requirements {
		var err error
		candidateSets[i], err = c.availableCandidates(ctx, req)
		if err != nil {
			return nil, false, err
		}
		if len(candidateSets[i]) == 0 {
			return nil, false, nil
		}
	}

	selected := make([]PickResult, len(requirements))
	reserved := make(map[string]measurement.Amount)
	var assign func(int) bool
	assign = func(index int) bool {
		if index == len(candidateSets) {
			return true
		}
		for _, pick := range candidateSets[index] {
			key := pick.IngredientID.String()
			prior, hadPrior := reserved[key]
			total := pick.Required
			if hadPrior {
				converted, err := prior.Convert(pick.Required.Unit())
				if err != nil {
					continue
				}
				total, err = converted.Add(pick.Required)
				if err != nil {
					continue
				}
			}
			available, err := pick.Available.Convert(total.Unit())
			if err != nil || available.Value() < total.Value() {
				continue
			}

			selected[index] = pick
			reserved[key] = total
			if assign(index + 1) {
				return true
			}
			if hadPrior {
				reserved[key] = prior
			} else {
				delete(reserved, key)
			}
		}
		return false
	}

	if !assign(0) {
		return nil, false, nil
	}
	return selected, true, nil
}

func (c *AvailabilityCalculator) availableCandidates(ctx store.Context, req drinksmodels.RecipeIngredient) ([]PickResult, error) {
	candidates := make([]candidate, 0, 1+len(req.Substitutes))
	candidates = append(candidates, candidate{
		id:            req.IngredientID,
		required:      req.Amount,
		isOriginal:    true,
		ratio:         1,
		qualityImpact: ingredientsmodels.QualityEquivalent,
	})

	seen := set.New(req.IngredientID.String())

	addCandidate := func(id entity.IngredientID, ratio float64, quality ingredientsmodels.Quality) {
		key := id.String()
		if seen.Contains(key) {
			return
		}
		seen.Add(key)
		candidates = append(candidates, candidate{
			id:            id,
			required:      req.Amount.Mul(ratio),
			isOriginal:    false,
			ratio:         ratio,
			qualityImpact: quality,
		})
	}

	var rules []ingredientsmodels.SubstitutionRule
	if c.ingredients != nil {
		resolved, err := c.ingredients.SubstitutionsFor(ctx, req.IngredientID)
		if err != nil && !errors.IsNotFound(err) {
			return nil, err
		}
		if err == nil {
			rules = resolved
		}
		sort.Slice(rules, func(i, j int) bool {
			if rules[i].QualityImpact.Rank() != rules[j].QualityImpact.Rank() {
				return rules[i].QualityImpact.Rank() > rules[j].QualityImpact.Rank()
			}
			return rules[i].SubstituteID.String() < rules[j].SubstituteID.String()
		})
	}

	rulesBySubstitute := make(map[string]ingredientsmodels.SubstitutionRule, len(rules))
	for _, rule := range rules {
		rulesBySubstitute[rule.SubstituteID.String()] = rule
	}
	for _, sub := range req.Substitutes {
		if rule, ok := rulesBySubstitute[sub.String()]; ok {
			addCandidate(rule.SubstituteID, rule.Ratio, rule.QualityImpact)
			continue
		}
		addCandidate(sub, 1.0, ingredientsmodels.QualitySimilar)
	}
	for _, rule := range rules {
		addCandidate(rule.SubstituteID, rule.Ratio, rule.QualityImpact)
	}

	var picks []PickResult
	for _, cand := range candidates {
		stock, err := c.inventory.Get(ctx, cand.id)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		available, err := stock.Available().Convert(cand.required.Unit())
		if err != nil {
			return nil, err
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
		return nil, nil
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

	if picks[0].Required.Value() <= 0 {
		return nil, nil
	}
	return picks, nil
}
