//nolint:paralleltest // terminal workflow request ordering intentionally runs serially.
package tui

import (
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/queries"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
)

func TestWorkflowResponsesIgnoreSupersededRequests(t *testing.T) {
	vm := NewListViewModel(nil)
	vm.workflowID = 2
	vm.mode = listModeAnalyzing
	vm.analysis = newAnalysisVM()

	vm.Update(analysisLoadedMsg{workflowID: 1, value: queries.MenuAnalytics{TotalCount: 99}})

	if vm.analysis.result != nil {
		t.Fatal("a superseded analysis response replaced the active workflow")
	}
}

func TestAnalysisTextDoesNotPresentUnknownCostAsKnown(t *testing.T) {
	cost := money.NewPriceFromCents(123, currency.USD)
	view := analysisText(queries.MenuAnalytics{Items: []queries.MenuItemAnalytics{{
		Name: "Unknown cost drink", Cost: &cost, CostUnknown: true,
	}}})

	if !strings.Contains(view, "Cost: unknown") {
		t.Fatalf("expected unknown cost marker, got:\n%s", view)
	}
	if strings.Contains(view, "$1.23") {
		t.Fatalf("unknown cost leaked a misleading amount:\n%s", view)
	}
}
