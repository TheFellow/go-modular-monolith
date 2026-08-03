package gui

// Command is an application-shell intent. A concrete surface decides whether
// the intent is meaningful in its current mode and how it maps to its widgets.
type Command string

const (
	CommandRefresh Command = "refresh"
	CommandNew     Command = "new"
	CommandSave    Command = "save"
	CommandCancel  Command = "cancel"
)

// Commander is implemented by framework-native views that expose keyboard
// actions to the shell. Returning false means the command was unavailable.
type Commander interface {
	ExecuteCommand(Command) bool
}

// Trigger invokes an enabled button through the same callback as pointer
// input. It keeps keyboard availability aligned with the visible control.
func Trigger(button *SemanticButton) bool {
	if button == nil || button.OnTapped == nil || button.Disabled() || button.Hidden {
		return false
	}
	button.OnTapped()
	return true
}

// SubmitOnEnter binds an entry to the same guarded action as its visible
// button. This prevents keyboard submission from bypassing hidden/disabled
// state or acquiring a second copy of the command callback.
func SubmitOnEnter(entry *SemanticEntry, button *SemanticButton) {
	entry.OnSubmitted = func(string) { Trigger(button) }
}
