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
	IngredientID  cedar.EntityUID
	SubstituteID  cedar.EntityUID
	Ratio         float64
	QualityImpact Quality
	Notes         string
}

func (r SubstitutionRule) Validate() error {
	if strings.TrimSpace(string(r.IngredientID.ID)) == "" {
		return errors.Invalidf("ingredient id is required")
	}
	if strings.TrimSpace(string(r.SubstituteID.ID)) == "" {
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

func DefaultSubstitutionRules() []SubstitutionRule {
	return []SubstitutionRule{
		{
			IngredientID:  entity.IngredientID("lime-juice"),
			SubstituteID:  entity.IngredientID("lemon-juice"),
			Ratio:         1.0,
			QualityImpact: QualitySimilar,
			Notes:         "Citrus swap; expect a slightly different profile",
		},
		{
			IngredientID:  entity.IngredientID("lemon-juice"),
			SubstituteID:  entity.IngredientID("lime-juice"),
			Ratio:         1.0,
			QualityImpact: QualitySimilar,
			Notes:         "Citrus swap; expect a slightly different profile",
		},
		{
			IngredientID:  entity.IngredientID("simple-syrup"),
			SubstituteID:  entity.IngredientID("honey-syrup"),
			Ratio:         0.75,
			QualityImpact: QualityDifferent,
			Notes:         "Honey is sweeter; reduce amount",
		},
		{
			IngredientID:  entity.IngredientID("bourbon"),
			SubstituteID:  entity.IngredientID("rye-whiskey"),
			Ratio:         1.0,
			QualityImpact: QualityEquivalent,
			Notes:         "Comparable spirit substitution",
		},
		{
			IngredientID:  entity.IngredientID("fresh-mint"),
			SubstituteID:  entity.IngredientID("dried-mint"),
			Ratio:         0.5,
			QualityImpact: QualityDifferent,
			Notes:         "Dried herbs are more concentrated",
		},
	}
}

func SubstitutionsFor(ingredientID cedar.EntityUID) []SubstitutionRule {
	out := make([]SubstitutionRule, 0, 2)
	for _, rule := range DefaultSubstitutionRules() {
		if string(rule.IngredientID.ID) == string(ingredientID.ID) {
			out = append(out, rule)
		}
	}
	return out
}

func LookupSubstitution(original cedar.EntityUID, substitute cedar.EntityUID) (SubstitutionRule, bool) {
	for _, rule := range DefaultSubstitutionRules() {
		if string(rule.IngredientID.ID) == string(original.ID) && string(rule.SubstituteID.ID) == string(substitute.ID) {
			return rule, true
		}
	}
	return SubstitutionRule{}, false
}
