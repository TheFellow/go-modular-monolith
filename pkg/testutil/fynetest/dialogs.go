package fynetest

import (
	"sync"

	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

type Confirmation struct {
	Title   string
	Message string
	Respond func(bool)
}

type Warning struct {
	Title   string
	Message string
}

type TaggedConfirmation struct {
	Title, Message, Current string
	Respond                 func(bool, ui.TagMutationMode, string)
}

// TaggedDialogs records the richer mutation confirmation without changing the
// basic Dialogs double used by tests that intentionally exercise preservation.
type TaggedDialogs struct {
	Dialogs
	taggedMu sync.Mutex
	tagged   []TaggedConfirmation
}

var _ ui.Dialogs = (*TaggedDialogs)(nil)
var _ ui.TaggedConfirmer = (*TaggedDialogs)(nil)

func (d *TaggedDialogs) ConfirmTagged(title, message, current string, respond func(bool, ui.TagMutationMode, string)) {
	d.taggedMu.Lock()
	d.tagged = append(d.tagged, TaggedConfirmation{Title: title, Message: message, Current: current, Respond: respond})
	d.taggedMu.Unlock()
}

func (d *TaggedDialogs) TaggedConfirmations() []TaggedConfirmation {
	d.taggedMu.Lock()
	defer d.taggedMu.Unlock()
	return append([]TaggedConfirmation(nil), d.tagged...)
}

type Dialogs struct {
	mu            sync.Mutex
	confirmations []Confirmation
	errors        []error
	warnings      []Warning
}

func (d *Dialogs) Confirm(title, message string, respond func(bool)) {
	d.mu.Lock()
	d.confirmations = append(d.confirmations, Confirmation{Title: title, Message: message, Respond: respond})
	d.mu.Unlock()
}

func (d *Dialogs) ShowError(err error) {
	d.mu.Lock()
	d.errors = append(d.errors, err)
	d.mu.Unlock()
}

func (d *Dialogs) ShowWarning(title, message string) {
	d.mu.Lock()
	d.warnings = append(d.warnings, Warning{Title: title, Message: message})
	d.mu.Unlock()
}

func (d *Dialogs) Confirmations() []Confirmation {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]Confirmation(nil), d.confirmations...)
}

func (d *Dialogs) Errors() []error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]error(nil), d.errors...)
}

func (d *Dialogs) Warnings() []Warning {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]Warning(nil), d.warnings...)
}
