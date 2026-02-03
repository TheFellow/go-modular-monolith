package tui

// Package-level styles and keys, computed once at init.
// ViewModels will import these directly in the next task.
var (
	appStyles = newStyles()
	appKeys   = newKeyMap()
)

// Pre-computed style/key subsets for ViewModels to import.
var (
	ListViewStyles = listViewStylesFrom(appStyles)
	ListViewKeys   = listViewKeysFrom(appKeys)
	FormStyles     = formStylesFrom(appStyles)
	FormKeys       = formKeysFrom(appKeys)
	DialogStyles   = dialogStylesFrom(appStyles)
	DialogKeys     = dialogKeysFrom(appKeys)
)

// AppStyles returns the full application styles (used by App for status bar, etc.).
func AppStyles() Styles { return appStyles }

// AppKeys returns the full application key map (used by App for global bindings).
func AppKeys() KeyMap { return appKeys }
