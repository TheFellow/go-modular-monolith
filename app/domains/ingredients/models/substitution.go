package models

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

type Quality string

const (
	QualityEquivalent Quality = "equivalent"
	QualitySimilar    Quality = "similar"
	QualityDifferent  Quality = "different"
)

func (q Quality) Rank() int {
	switch q {
	case QualityEquivalent:
		return 3
	case QualitySimilar:
		return 2
	case QualityDifferent:
		return 1
	default:
		return 0
	}
}

func (q Quality) Validate() error {
	switch q {
	case QualityEquivalent, QualitySimilar, QualityDifferent:
		return nil
	default:
		return errors.Invalidf("invalid quality %q", string(q))
	}
}

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
			IngredientID:  NewIngredientID("lime-juice"),
			SubstituteID:  NewIngredientID("lemon-juice"),
			Ratio:         1.0,
			QualityImpact: QualitySimilar,
			Notes:         "Citrus swap; expect a slightly different profile",
		},
		{
			IngredientID:  NewIngredientID("lemon-juice"),
			SubstituteID:  NewIngredientID("lime-juice"),
			Ratio:         1.0,
			QualityImpact: QualitySimilar,
			Notes:         "Citrus swap; expect a slightly different profile",
		},
		{
			IngredientID:  NewIngredientID("simple-syrup"),
			SubstituteID:  NewIngredientID("honey-syrup"),
			Ratio:         0.75,
			QualityImpact: QualityDifferent,
			Notes:         "Honey is sweeter; reduce amount",
		},
		{
			IngredientID:  NewIngredientID("bourbon"),
			SubstituteID:  NewIngredientID("rye-whiskey"),
			Ratio:         1.0,
			QualityImpact: QualityEquivalent,
			Notes:         "Comparable spirit substitution",
		},
		{
			IngredientID:  NewIngredientID("fresh-mint"),
			SubstituteID:  NewIngredientID("dried-mint"),
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
