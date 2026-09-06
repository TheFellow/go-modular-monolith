package models

import (
	"math"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/quality"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type Quality = quality.Quality

const (
	QualityEquivalent = quality.Equivalent
	QualitySimilar    = quality.Similar
	QualityDifferent  = quality.Different
)

type SubstitutionRule struct {
	Revision      uint64 `json:"revision"`
	Disabled      bool
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
	if r.Ratio <= 0 || math.IsNaN(r.Ratio) || math.IsInf(r.Ratio, 0) {
		return errors.Invalidf("ratio must be > 0")
	}
	if err := r.QualityImpact.Validate(); err != nil {
		return err
	}
	return nil
}
