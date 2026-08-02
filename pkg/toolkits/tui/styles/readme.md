# TUI styles

`styles` defines one reusable Lip Gloss theme and the derived style subsets consumed by list/detail
views, forms, dialogs, and dashboards. `Standard` is the ready-to-use default; callers can retain a
`Styles` value when they need to pass the complete theme through a shell.

The package owns visual mechanics and color defaults, not domain status meaning. Domain adapters
choose the appropriate style for labels such as draft, published, low stock, or cancelled.
