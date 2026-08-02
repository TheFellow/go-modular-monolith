# TUI presentation helpers

This package contains small, domain-neutral display transformations shared by terminal adapters.
`LabelOr` currently selects caller-provided fallback text for an empty value. The caller owns both
the value's meaning and the wording; this package only centralizes the recurring transformation.

Keep domain labels and formatting beside their domain adapter. Add helpers here only when their
contract remains useful without importing application models.
