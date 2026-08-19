package menus

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

type readinessResult struct {
	menu   *models.Menu
	report models.ReadinessReport
}

func (r readinessResult) CedarEntity() cedar.Entity { return r.menu.CedarEntity() }

func (m *Module) Readiness(ctx *middleware.Context, id entity.MenuID) (models.ReadinessReport, error) {
	result, err := m.pipeline.Query(ctx, authz.ActionReadiness, id,
		func(ctx store.Context, id entity.MenuID) (readinessResult, error) {
			menu, report, err := m.queries.Readiness(ctx, id)
			return readinessResult{menu: menu, report: report}, err
		})
	return result.report, err
}
