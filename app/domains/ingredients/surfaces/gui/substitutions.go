package gui

import (
	"context"
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	toolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

type substitutionResult struct {
	rules []models.SubstitutionRule
	names map[entity.IngredientID]string
}

type SubstitutionForm struct {
	SubstituteID, Ratio, Notes string
	Quality                    models.Quality
	Disabled                   bool
	Revision                   uint64
}

func (p *Presenter) StartSubstitutions() {
	p.mu.Lock()
	if p.state.Submitting || p.state.Dirty || p.state.Selected == nil || !p.actionEnabledLocked(ingredients.ControlSubstitutions) {
		p.mu.Unlock()
		return
	}
	p.state.Mode, p.state.Err = Substitutions, nil
	p.state.Rules = nil
	p.state.RuleForm = SubstitutionForm{Ratio: "1", Quality: models.QualityEquivalent}
	p.state.FormInstance++
	p.publishLocked()
	p.mu.Unlock()
	p.LoadSubstitutions()
}

func (p *Presenter) LoadSubstitutions() {
	p.mu.Lock()
	if p.state.Selected == nil || p.state.Mode != Substitutions || p.state.Submitting {
		p.mu.Unlock()
		return
	}
	id := p.state.Selected.ID
	p.mu.Unlock()
	p.ruleLoads.LoadContext(p.app.Context(), func(ctx context.Context) (substitutionResult, error) {
		rules, err := p.app.Ingredients.SubstitutionRules(p.app.ContextFrom(ctx), id)
		if err != nil {
			return substitutionResult{}, err
		}
		names := make(map[entity.IngredientID]string, len(rules))
		for _, rule := range rules {
			names[rule.SubstituteID] = rule.SubstituteID.String()
			if substitute, err := p.app.Ingredients.Get(p.app.ContextFrom(ctx), rule.SubstituteID); err == nil {
				names[rule.SubstituteID] = substitute.Name
			}
		}
		return substitutionResult{rules: rules, names: names}, nil
	}, func(result toolkit.LoadState[substitutionResult]) {
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.state.Mode != Substitutions || selectedID(p.state.Selected) != id {
			return
		}
		p.state.RuleStatus = result.Status
		p.state.Err = toolkit.PresentError(result.Err)
		if result.Status == toolkit.Loaded {
			p.state.Rules, p.state.RuleNames = result.Value.rules, result.Value.names
		}
		p.publishLocked()
	})
}

func (p *Presenter) SelectSubstitution(id entity.IngredientID) {
	p.selectSubstitution(id, false)
}

func (p *Presenter) selectSubstitution(id entity.IngredientID, discard bool) {
	p.mu.Lock()
	if p.state.Submitting {
		p.mu.Unlock()
		return
	}
	if p.state.Dirty && !discard {
		p.mu.Unlock()
		if p.dialogs != nil {
			p.dialogs.Confirm("Discard changes?", "Discard unsaved substitution rule changes?", func(ok bool) {
				if ok {
					p.selectSubstitution(id, true)
				} else {
					p.mu.Lock()
					p.publishLocked()
					p.mu.Unlock()
				}
			})
		}
		return
	}
	defer p.mu.Unlock()
	form := SubstitutionForm{Ratio: "1", Quality: models.QualityEquivalent}
	for _, rule := range p.state.Rules {
		if rule.SubstituteID == id {
			form = SubstitutionForm{SubstituteID: id.String(), Ratio: strconv.FormatFloat(rule.Ratio, 'g', -1, 64), Quality: rule.QualityImpact, Notes: rule.Notes, Disabled: rule.Disabled, Revision: rule.Revision}
			break
		}
	}
	p.state.RuleForm, p.state.Dirty, p.state.Err = form, false, nil
	p.state.FormInstance++
	p.publishLocked()
}

func (p *Presenter) SetSubstitutionForm(form SubstitutionForm) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// Stable identity and revision come from the selected persisted rule.
	if p.state.RuleForm.Revision > 0 {
		form.SubstituteID = p.state.RuleForm.SubstituteID
	}
	form.Revision = p.state.RuleForm.Revision
	p.state.Dirty = p.state.Dirty || form != p.state.RuleForm
	p.state.RuleForm = form
	p.publishLocked()
}

func (p *Presenter) SubmitSubstitution() bool {
	p.mu.Lock()
	if p.state.Mode != Substitutions || p.state.Selected == nil || p.state.Submitting || !p.actionEnabledLocked(ingredients.ControlSetSubstitution) {
		p.mu.Unlock()
		return false
	}
	form := p.state.RuleForm
	id, err := entity.ParseIngredientID(strings.TrimSpace(form.SubstituteID))
	ratio, ratioErr := strconv.ParseFloat(strings.TrimSpace(form.Ratio), 64)
	if err == nil && ratioErr != nil {
		err = errors.Invalidf("ratio must be a positive number")
	}
	rule := models.SubstitutionRule{IngredientID: p.state.Selected.ID, SubstituteID: id, Ratio: ratio, QualityImpact: form.Quality, Notes: strings.TrimSpace(form.Notes), Disabled: form.Disabled, Revision: form.Revision}
	if err == nil {
		err = rule.Validate()
	}
	if err != nil {
		p.state.Err = toolkit.PresentError(err)
		p.publishLocked()
		p.mu.Unlock()
		return false
	}
	p.state.Submitting, p.state.Err = true, nil
	p.publishLocked()
	p.mu.Unlock()
	accepted := p.mutation.Submit(func() error { _, err := p.app.Ingredients.SetSubstitution(p.app.Context(), &rule); return err }, func(err error) {
		p.mu.Lock()
		p.state.Submitting, p.state.Err = false, toolkit.PresentError(err)
		if err == nil {
			p.state.Dirty = false
			p.state.RuleForm = SubstitutionForm{Ratio: "1", Quality: models.QualityEquivalent}
			p.state.FormInstance++
		}
		p.publishLocked()
		p.mu.Unlock()
		toolkit.ShowPresentation(p.dialogs, err)
		// A conflict retains the draft and its stale revision. Refresh and select the
		// persisted rule explicitly before reconciling, never silently overwrite it.
		if err == nil {
			p.LoadSubstitutions()
		}
	})
	if !accepted {
		p.mu.Lock()
		p.state.Submitting = p.mutation.Active()
		p.publishLocked()
		p.mu.Unlock()
	}
	return accepted
}
