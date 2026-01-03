package cli

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
)

func ValidateCategory(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return models.Category(s).Validate()
}

func ValidateUnit(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return models.Unit(s).Validate()
}
