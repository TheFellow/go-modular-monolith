package gui

import (
	"fmt"
	"slices"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	presentation "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces"
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

const (
	ControlAmend              = "orders-amend"
	ControlAmendReason        = "orders-amend-reason"
	ControlAmendSave          = "orders-amend-save"
	ControlAmendQueue         = "orders-amend-queue"
	ControlAmendBatch         = "orders-amend-batch"
	ControlAmendBatchSave     = "orders-amend-batch-save"
	ControlAmendBatchClear    = "orders-amend-batch-clear"
	ControlCancellationReason = "orders-cancellation-reason"
)

func (p *Presenter) StartAmend() {
	if p.busy() || p.state.Mode != Viewing || p.state.Selected == nil || !actionEnabled(p.state.Actions, orders.ControlAmend) {
		return
	}
	p.state.Amendment = presentation.NewAmendmentForm(p.state.Selected.Order)
	p.state.Mode, p.state.Dirty, p.state.Err = Amending, false, nil
	p.publish()
}
func (p *Presenter) SetAmendment(form presentation.AmendmentForm) {
	if p.state.Mode != Amending || p.state.Submitting {
		return
	}
	// Target identity and expected revision belong to the opened editor.
	form.OrderID, form.Revision = p.state.Amendment.OrderID, p.state.Amendment.Revision
	p.state.Amendment, p.state.Dirty = form.Clone(), true
}
func (p *Presenter) SaveAmendment(queue bool) bool {
	if p.state.Mode != Amending || p.busy() || !actionEnabled(p.state.Actions, orders.ControlAmend) {
		return false
	}
	request, err := p.state.Amendment.Request()
	if err != nil {
		p.fail(err)
		return false
	}
	if queue {
		p.state.AmendmentQueue = presentation.Queue(p.state.AmendmentQueue, request)
		p.state.Mode, p.state.Dirty = Browsing, false
		p.publish()
		return true
	}
	return p.mutate(func() error { _, err := p.app.Orders.Amend(p.app.Context(), request); return err }, true)
}
func (p *Presenter) ReviewAmendments() {
	if p.busy() || len(p.state.AmendmentQueue) == 0 {
		return
	}
	p.state.Mode, p.state.Err = ReviewingBatch, nil
	p.publish()
}
func (p *Presenter) ClearAmendments() {
	if p.busy() {
		return
	}
	p.state.AmendmentQueue, p.state.Mode, p.state.Err = nil, Browsing, nil
	p.publish()
}
func (p *Presenter) SaveAmendmentBatch() bool {
	if p.busy() || p.state.Mode != ReviewingBatch || len(p.state.AmendmentQueue) == 0 {
		return false
	}
	requests := slices.Clone(p.state.AmendmentQueue)
	p.state.Submitting, p.state.Err = true, nil
	p.publish()
	return p.submit.Submit(func() error { _, err := p.app.Orders.AmendBatch(p.app.Context(), requests); return err }, func(err error) {
		p.state.Submitting, p.state.Err = false, ui.PresentError(err)
		if err == nil {
			p.state.AmendmentQueue, p.state.Mode = nil, Browsing
		}
		p.publish()
		ui.ShowPresentation(p.dialogs, err)
		if err == nil {
			p.Refresh()
		}
	})
}

func (v *View) amendmentForm(s State) framework.CanvasObject {
	form := s.Amendment.Clone()
	fields := container.NewVBox(widget.NewLabel(fmt.Sprintf("Order %s · revision %d", form.OrderID, form.Revision)))
	reason := ui.NewMultiLineEntry(ControlAmendReason)
	reason.SetText(form.Reason)
	reason.OnChanged = func(value string) { form.Reason = value; v.presenter.SetAmendment(form) }
	fields.Add(ui.DetailField("Reason", reason))
	for i, field := range form.Replacements {
		replacement := ui.NewEntry(fmt.Sprintf("orders-amend-replacement-%d", i))
		replacement.SetPlaceHolder("Replacement ingredient ID (blank keeps current)")
		replacement.SetText(field.ReplacementID)
		replacement.OnChanged = func(value string) { form.Replacements[i].ReplacementID = value; v.presenter.SetAmendment(form) }
		ratio := ui.NewEntry(fmt.Sprintf("orders-amend-ratio-%d", i))
		ratio.SetText(field.Ratio)
		ratio.OnChanged = func(value string) { form.Replacements[i].Ratio = value; v.presenter.SetAmendment(form) }
		setEnabled(replacement, !s.Submitting)
		setEnabled(ratio, !s.Submitting)
		fields.Add(ui.FormSection(field.Name, "Current ingredient: "+field.ID.String(), ui.DetailField("Replacement ingredient", replacement), ui.DetailField("Quantity ratio", ratio)))
	}
	for i, field := range form.Preparation {
		steps := ui.NewMultiLineEntry(fmt.Sprintf("orders-amend-steps-%d", i))
		steps.SetText(field.Steps)
		steps.OnChanged = func(value string) { form.Preparation[i].Steps = value; v.presenter.SetAmendment(form) }
		garnish := ui.NewEntry(fmt.Sprintf("orders-amend-garnish-%d", i))
		garnish.SetText(field.Garnish)
		garnish.OnChanged = func(value string) { form.Preparation[i].Garnish = value; v.presenter.SetAmendment(form) }
		setEnabled(steps, !s.Submitting)
		setEnabled(garnish, !s.Submitting)
		fields.Add(ui.FormSection(field.Name+" preparation", "One step per line. Clear garnish to remove it.", ui.DetailField("Steps", steps), ui.DetailField("Garnish", garnish)))
	}
	v.save = ui.NewButton(ControlAmendSave, "Approve amendment", func() { v.presenter.SaveAmendment(false) })
	queue := ui.NewButton(ControlAmendQueue, "Add to batch", func() { v.presenter.SaveAmendment(true) })
	v.cancel = ui.NewButton(ControlFormCancel, "Cancel", v.presenter.CancelForm)
	setEnabled(v.save, !s.Submitting)
	setEnabled(queue, !s.Submitting)
	setEnabled(v.cancel, !s.Submitting)
	setEnabled(reason, !s.Submitting)
	fields.Add(queue)
	message := ""
	if s.Err != nil {
		message = "Error: " + s.Err.Error()
	}
	return ui.StandardFormPage(ui.FormPage{Title: "Amend order", Breadcrumb: v.breadcrumb("Amend order"), Subtitle: "Approve ingredient replacements and preparation while preserving original acceptance.", Fields: fields, Status: widget.NewLabel(message), Save: v.save, Cancel: v.cancel})
}
func (v *View) amendmentBatch(s State) framework.CanvasObject {
	fields := container.NewVBox(ui.ReadonlyMultiLineEntry(presentation.BatchSummary(s.AmendmentQueue)))
	clearBatch := ui.NewButton(ControlAmendBatchClear, "Clear batch", v.presenter.ClearAmendments)
	v.save = ui.NewButton(ControlAmendBatchSave, "Approve all amendments", func() { v.presenter.SaveAmendmentBatch() })
	v.cancel = ui.NewButton(ControlFormCancel, "Back", v.presenter.Back)
	setEnabled(v.save, !s.Submitting)
	setEnabled(clearBatch, !s.Submitting)
	setEnabled(v.cancel, !s.Submitting)
	fields.Add(clearBatch)
	message := ""
	if s.Err != nil {
		message = "Error: " + s.Err.Error()
	}
	return ui.StandardFormPage(ui.FormPage{Title: "Review amendment batch", Subtitle: "All amendments succeed together. If an order changed or stock is insufficient, none are applied. Reopen an order to replace its queued amendment.", Fields: fields, Save: v.save, Cancel: v.cancel, Status: widget.NewLabel(message)})
}

func cloneAmendments(requests []models.Amendment) []models.Amendment {
	out := slices.Clone(requests)
	for i := range out {
		out[i].Replacements = slices.Clone(out[i].Replacements)
		out[i].Preparation = slices.Clone(out[i].Preparation)
		for j := range out[i].Preparation {
			out[i].Preparation[j].Steps = slices.Clone(out[i].Preparation[j].Steps)
		}
	}
	return out
}

func (p *Presenter) SetCancellationReason(value string) {
	if p.state.Mode != Viewing || p.busy() || !p.state.CanCancel {
		return
	}
	p.state.CancellationReason = value
}
