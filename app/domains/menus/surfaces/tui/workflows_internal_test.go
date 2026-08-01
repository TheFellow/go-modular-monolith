//nolint:paralleltest // terminal workflow request ordering intentionally runs serially.
package tui

import (
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/queries"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestWorkflowResponsesIgnoreSupersededRequests(t *testing.T) {
	vm := NewListViewModel(nil)
	vm.workflowID = 2
	vm.mode = listModeAnalyzing
	vm.analysis = newAnalysisVM()

	vm.Update(analysisLoadedMsg{workflowID: 1, value: queries.MenuAnalytics{TotalCount: 99}})

	testutil.ErrorIf(t, vm.analysis.result != nil, "%v", "a superseded analysis response replaced the active workflow")
}

func TestAnalysisTextDoesNotPresentUnknownCostAsKnown(t *testing.T) {
	cost := money.NewPriceFromCents(123, currency.USD)
	view := analysisText(queries.MenuAnalytics{Items: []queries.MenuItemAnalytics{{
		Name: "Unknown cost drink", Cost: &cost, CostUnknown: true,
	}}})

	testutil.ErrorIf(t, !strings.Contains(view, "Cost: unknown"), "expected unknown cost marker, got:\n%s", view)
	testutil.ErrorIf(t, strings.Contains(view, "$1.23"), "unknown cost leaked a misleading amount:\n%s", view)
}
