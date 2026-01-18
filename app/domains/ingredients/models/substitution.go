package models

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/quality"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

type Quality = quality.Quality

const (
	QualityEquivalent = quality.Equivalent
	QualitySimilar    = quality.Similar
	QualityDifferent  = quality.Different
)

type SubstitutionRule struct {
	IngredientID  entity.IngredientID
	SubstituteID  entity.IngredientID
	Ratio         float64
	QualityImpact Quality
	Notes         string
}

func (r SubstitutionRule) Validate() error {
	if strings.TrimSpace(r.IngredientID.String()) == "" {
		return errors.Invalidf("ingredient id is required")
	}
	if strings.TrimSpace(r.SubstituteID.String()) == "" {
		return errors.Invalidf("substitute id is required")
	}
	if r.Ratio <= 0 {
		return errors.Invalidf("ratio must be > 0")
	}
	if err := r.QualityImpact.Validate(); err != nil {
		return err
	}
	return nil
}

func newIngredientID(id string) entity.IngredientID {
	return entity.IngredientID(cedar.NewEntityUID(entity.TypeIngredient, cedar.String(id)))
}

func DefaultSubstitutionRules() []SubstitutionRule {
	return []SubstitutionRule{
		{
			IngredientID:  newIngredientID("lime-juice"),
			SubstituteID:  newIngredientID("lemon-juice"),
			Ratio:         1.0,
			QualityImpact: QualitySimilar,
			Notes:         "Citrus swap; expect a slightly different profile",
		},
		{
			IngredientID:  newIngredientID("lemon-juice"),
			SubstituteID:  newIngredientID("lime-juice"),
			Ratio:         1.0,
			QualityImpact: QualitySimilar,
			Notes:         "Citrus swap; expect a slightly different profile",
		},
		{
			IngredientID:  newIngredientID("simple-syrup"),
			SubstituteID:  newIngredientID("honey-syrup"),
			Ratio:         0.75,
			QualityImpact: QualityDifferent,
			Notes:         "Honey is sweeter; reduce amount",
		},
		{
			IngredientID:  newIngredientID("bourbon"),
			SubstituteID:  newIngredientID("rye-whiskey"),
			Ratio:         1.0,
			QualityImpact: QualityEquivalent,
			Notes:         "Comparable spirit substitution",
		},
		{
			IngredientID:  newIngredientID("fresh-mint"),
			SubstituteID:  newIngredientID("dried-mint"),
			Ratio:         0.5,
			QualityImpact: QualityDifferent,
			Notes:         "Dried herbs are more concentrated",
		},
	}
}

func SubstitutionsFor(ingredientID entity.IngredientID) []SubstitutionRule {
	out := make([]SubstitutionRule, 0, 2)
	for _, rule := range DefaultSubstitutionRules() {
		if rule.IngredientID.String() == ingredientID.String() {
			out = append(out, rule)
		}
	}
	return out
}

func LookupSubstitution(original entity.IngredientID, substitute entity.IngredientID) (SubstitutionRule, bool) {
	for _, rule := range DefaultSubstitutionRules() {
		if rule.IngredientID.String() == original.String() && rule.SubstituteID.String() == substitute.String() {
			return rule, true
		}
	}
	return SubstitutionRule{}, false
}
