package menus

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/queries"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

// Analyze calculates operational details for an already-authorized menu.
func (m *Module) Analyze(ctx *middleware.Context, menu models.Menu, targetMargin float64) (queries.MenuAnalytics, error) {
	return m.analytics.Analyze(ctx, menu, targetMargin)
}
