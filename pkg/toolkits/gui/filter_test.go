package gui

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestFilterBarPresetsWriteVisibleDomainExpression(t *testing.T) {
	startTestApp(t)
	applied := ""
	bar := NewFilterBar("filter", "apply", "Filter…", `tags contains "featured"`, []FilterPreset{{
		ID: "status", Placeholder: "Status", Options: []FilterOption{{Label: "Pending", Expression: `status == "pending"`}, {Label: "Complete", Expression: `status == "completed"`}},
	}}, nil, widget.NewLabel("Page size"), func(value string) { applied = value })
	bar.Presets[0].SetSelected("Pending")
	if got, want := bar.Expression.Text, `tags contains "featured" && status == "pending"`; got != want {
		t.Fatalf("expression = %q, want %q", got, want)
	}
	bar.Presets[0].SetSelected("Complete")
	if got, want := bar.Expression.Text, `tags contains "featured" && status == "completed"`; got != want {
		t.Fatalf("expression = %q, want %q", got, want)
	}
	test.Tap(bar.Apply)
	if applied != bar.Expression.Text {
		t.Fatalf("applied %q, want %q", applied, bar.Expression.Text)
	}
	if bar.Advanced == nil || bar.Advanced.Items[0].Open {
		t.Fatal("advanced filters should start collapsed")
	}
}
