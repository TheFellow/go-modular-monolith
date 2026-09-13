package gui

import (
	"fmt"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

const (
	ControlSubstitutions  = string(ingredients.ControlSubstitutions)
	ControlRuleNew        = "ingredient-rule-new"
	ControlRuleSave       = "ingredient-rule-save"
	ControlRuleRefresh    = "ingredient-rule-refresh"
	ControlRuleSubstitute = "ingredient-rule-substitute"
	ControlRuleRatio      = "ingredient-rule-ratio"
	ControlRuleNotes      = "ingredient-rule-notes"
)

type substitutionView struct {
	panel                    *framework.Container
	rows                     *widget.List
	substitute, ratio, notes *ui.SemanticEntry
	quality                  *widget.Select
	disabled                 *widget.Check
	save, create             *ui.SemanticButton
	status, revision         *widget.Label
	instance                 uint64
}

func (v *View) newSubstitutionsPanel() *substitutionView {
	p := v.presenter
	r := &substitutionView{}
	r.substitute, r.ratio, r.notes = ui.NewEntry(ControlRuleSubstitute), ui.NewEntry(ControlRuleRatio), ui.NewEntry(ControlRuleNotes)
	r.substitute.PlaceHolder = "Active substitute ingredient ID"
	r.quality = widget.NewSelect([]string{"equivalent", "similar", "different"}, nil)
	r.disabled = widget.NewCheck("Disabled", nil)
	r.revision, r.status = widget.NewLabel(""), widget.NewLabel("")
	r.rows = widget.NewList(func() int { return len(v.state.Rules) }, func() framework.CanvasObject {
		label := widget.NewLabel("\n\n")
		label.Truncation = framework.TextTruncateEllipsis
		return label
	}, func(id widget.ListItemID, object framework.CanvasObject) {
		rule := v.state.Rules[id]
		status := "enabled"
		if rule.Disabled {
			status = "disabled"
		}
		object.(*widget.Label).SetText(fmt.Sprintf("%s\nRatio %g · %s\n%s · revision %d", v.state.RuleNames[rule.SubstituteID], rule.Ratio, rule.QualityImpact, status, rule.Revision))
	})
	r.rows.OnSelected = func(id widget.ListItemID) {
		if !v.rendering {
			p.SelectSubstitution(v.state.Rules[id].SubstituteID)
		}
	}
	r.save = ui.WithIcon(ui.NewButton(ControlRuleSave, "Save rule", func() { p.SubmitSubstitution() }), ui.IconSave)
	r.create = ui.WithIcon(ui.NewButton(ControlRuleNew, "New rule", func() { p.SelectSubstitution(entity.IngredientID{}) }), ui.IconAdd)
	refresh := ui.WithIcon(ui.NewButton(ControlRuleRefresh, "Refresh rules", p.LoadSubstitutions), ui.IconRefresh)
	fields := ui.DetailForm(ui.DetailField("Substitute ID", r.substitute), ui.DetailField("Ratio", r.ratio), ui.DetailField("Quality impact", r.quality), ui.DetailField("Notes", r.notes), ui.DetailField("Status", r.disabled), ui.DetailField("Revision", r.revision))
	editor := ui.StandardFormPage(ui.FormPage{Title: "Substitution rule", Subtitle: "Select a rule to revise, disable, or re-enable it. Refresh and reselect after a conflict.", Fields: fields, Status: r.status, Save: r.save, Cancel: ui.NewButton("ingredient-rule-back", "Back", p.Back)})
	r.panel = ui.StandardListPage(ui.ListPage{Title: "Ingredient substitutions", CollectionActions: []framework.CanvasObject{r.create, refresh}, List: r.rows, Detail: editor, ListRatio: .35}).(*framework.Container)
	change := func() {
		if v.rendering {
			return
		}
		p.SetSubstitutionForm(SubstitutionForm{SubstituteID: r.substitute.Text, Ratio: r.ratio.Text, Notes: r.notes.Text, Quality: models.Quality(r.quality.Selected), Disabled: r.disabled.Checked})
	}
	r.substitute.OnChanged, r.ratio.OnChanged, r.notes.OnChanged = func(string) { change() }, func(string) { change() }, func(string) { change() }
	r.quality.OnChanged = func(string) { change() }
	r.disabled.OnChanged = func(bool) { change() }
	return r
}

func (v *View) renderSubstitutions(s State) {
	r := v.rules
	r.panel.Hidden = s.Mode != Substitutions
	if s.Mode != Substitutions {
		return
	}
	if r.instance != s.FormInstance {
		r.instance = s.FormInstance
		if s.RuleForm.Revision == 0 {
			r.rows.UnselectAll()
		}
		r.substitute.SetText(s.RuleForm.SubstituteID)
		r.ratio.SetText(s.RuleForm.Ratio)
		r.notes.SetText(s.RuleForm.Notes)
		r.quality.SetSelected(string(s.RuleForm.Quality))
		r.disabled.SetChecked(s.RuleForm.Disabled)
	}
	for i, rule := range s.Rules {
		if rule.Revision > 0 && rule.SubstituteID.String() == s.RuleForm.SubstituteID {
			r.rows.Select(i)
			break
		}
	}
	r.revision.SetText(fmt.Sprintf("%d", s.RuleForm.Revision))
	writable := s.Actions[ingredients.ControlSetSubstitution].Enabled && !s.Submitting
	r.create.Hidden, r.save.Hidden = !s.Actions[ingredients.ControlSetSubstitution].Visible, !s.Actions[ingredients.ControlSetSubstitution].Visible
	r.substitute.Disable()
	r.ratio.Disable()
	r.notes.Disable()
	r.quality.Disable()
	r.disabled.Disable()
	r.save.Disable()
	r.create.Disable()
	if writable {
		if s.RuleForm.Revision == 0 {
			r.substitute.Enable()
		}
		r.ratio.Enable()
		r.notes.Enable()
		r.quality.Enable()
		r.disabled.Enable()
		r.save.Enable()
		r.create.Enable()
	}
	switch {
	case s.Err != nil:
		r.status.SetText("Error: " + s.Err.Error())
	case s.Submitting:
		r.status.SetText("Saving rule…")
	case s.RuleStatus == ui.Loading:
		r.status.SetText("Loading rules…")
	default:
		r.status.SetText(fmt.Sprintf("%d rules (including disabled)", len(s.Rules)))
	}
	r.rows.Refresh()
}
