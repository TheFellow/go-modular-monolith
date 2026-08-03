//nolint:paralleltest // Fyne's application driver and focus state are process-global.
package gui

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestFilterBarPresetsWriteVisibleDomainExpression(t *testing.T) {
	startTestApp(t)
	applied := ""
	bar := NewFilterBar("filter", "apply", "Filter…", `tags contains "featured"`, []FilterPreset{{
		ID: "status", Placeholder: "Status", Options: []FilterOption{{Label: "Pending", Expression: `status == "pending"`}, {Label: "Complete", Expression: `status == "completed"`}},
	}}, nil, widget.NewLabel("Page size"), func(value string) { applied = value })
	bar.Presets[0].SetSelected("Pending")
	{
		got, want := bar.Expression.Text, `tags contains "featured" && status == "pending"`
		testutil.ErrorIf(t, got != want, "expression = %q, want %q", got, want)
	}
	bar.Presets[0].SetSelected("Complete")
	{
		got, want := bar.Expression.Text, `tags contains "featured" && status == "completed"`
		testutil.ErrorIf(t, got != want, "expression = %q, want %q", got, want)
	}
	test.Tap(bar.Apply)
	testutil.ErrorIf(t, applied != bar.Expression.Text, "applied %q, want %q", applied, bar.Expression.Text)
	testutil.ErrorIf(t, bar.Advanced == nil || bar.Advanced.Items[0].Open, "%v", "advanced filters should start collapsed")
}

func TestFilterBarEnterAppliesTrimmedExpression(t *testing.T) {
	startTestApp(t)
	applied := ""
	bar := NewSingleRowFilterBar("filter", "apply", "Filter…", "", nil, nil, func(value string) { applied = value })
	bar.Expression.SetText("  name == \"Negroni\"  ")
	bar.Expression.OnSubmitted(bar.Expression.Text)
	testutil.ErrorIf(t, applied != `name == "Negroni"`, "Enter applied %q", applied)
}

func TestFilterBarEnterCannotBypassDisabledState(t *testing.T) {
	startTestApp(t)
	calls := 0
	bar := NewSingleRowFilterBar("filter", "apply", "Filter…", "", nil, nil, func(string) { calls++ })
	bar.SetEnabled(false)
	bar.Expression.OnSubmitted("ignored")
	bar.SetEnabled(true)
	bar.Expression.OnSubmitted("applied")
	testutil.ErrorIf(t, calls != 1, "filter applied %d times, want only enabled submission", calls)
}
