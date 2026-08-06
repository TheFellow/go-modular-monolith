package tui

import (
	"testing"

	menusdomain "github.com/TheFellow/go-modular-monolith/app/domains/menus"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	"github.com/charmbracelet/bubbles/list"
)

func TestDetailReadinessIgnoresStaleSelection(t *testing.T) {
	t.Parallel()
	first := models.Menu{ID: models.NewMenuID("first"), Name: "First", Status: models.MenuStatusDraft}
	second := models.Menu{ID: models.NewMenuID("second"), Name: "Second", Status: models.MenuStatusDraft}
	vm := &ListViewModel{
		list:    list.New([]list.Item{newMenuItem(second, styles.Standard.ListView)}, list.NewDefaultDelegate(), 80, 20),
		detail:  NewDetailViewModel(styles.Standard.ListView, nil),
		actions: map[actions.ID]actions.State{menusdomain.ControlPublish: {ID: menusdomain.ControlPublish, Visible: true, Enabled: true}},
	}
	report := &models.ReadinessReport{Findings: []models.ReadinessFinding{{Severity: models.ReadinessBlocker}}}
	_, _ = vm.Update(detailLoadedMsg{menuID: first.ID, readiness: report})
	testutil.Equals(t, vm.detail.readiness == nil, true)
	_, _ = vm.Update(detailLoadedMsg{menuID: second.ID, readiness: report})
	testutil.Equals(t, vm.detail.readiness == report, true)
	testutil.Equals(t, vm.actions[menusdomain.ControlPublish].Enabled, false)
}
