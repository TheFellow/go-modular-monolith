package models

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type ReadinessSeverity string

const (
	ReadinessBlocker ReadinessSeverity = "blocker"
	ReadinessWarning ReadinessSeverity = "warning"
)

type ReadinessCode string

const (
	ReadinessReviewRequired        ReadinessCode = "review_required_drink"
	ReadinessRetiredIngredient     ReadinessCode = "retired_or_missing_ingredient"
	ReadinessTemporarySubstitution ReadinessCode = "temporary_substitution"
	ReadinessUnavailable           ReadinessCode = "unavailable"
	ReadinessLowStock              ReadinessCode = "low_stock"
)

type ReadinessFinding struct {
	Severity     ReadinessSeverity
	Code         ReadinessCode
	DrinkID      entity.DrinkID
	IngredientID entity.IngredientID
	Message      string
}

type ReadinessReport struct {
	MenuID   entity.MenuID
	Status   MenuStatus
	Findings []ReadinessFinding
}

func (r ReadinessReport) HasBlockers() bool {
	for _, finding := range r.Findings {
		if finding.Severity == ReadinessBlocker {
			return true
		}
	}
	return false
}

func (r ReadinessReport) RequireReady() error {
	if !r.HasBlockers() {
		return nil
	}
	messages := make([]string, 0, len(r.Findings))
	for _, finding := range r.Findings {
		if finding.Severity == ReadinessBlocker {
			messages = append(messages, finding.Message)
		}
	}
	return errors.FailedPreconditionf("menu %q is not ready to publish: %s", r.MenuID.String(), strings.Join(messages, "; "))
}
