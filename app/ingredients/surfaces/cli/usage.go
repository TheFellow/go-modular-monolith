package cli

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
)

func CategoryUsage() string {
	return "Category (" + strings.Join(categoryOptions(), "|") + ")"
}

func categoryOptions() []string {
	supported := models.AllCategories()
	out := make([]string, 0, len(supported))
	for _, c := range supported {
		out = append(out, string(c))
	}
	return out
}

func UnitUsage() string {
	return "Unit (" + strings.Join(unitOptions(), "|") + ")"
}

func unitOptions() []string {
	supported := models.AllUnits()
	out := make([]string, 0, len(supported))
	for _, u := range supported {
		out = append(out, string(u))
	}
	return out
}
