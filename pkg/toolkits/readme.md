# Presentation toolkits

`pkg/toolkits` contains reusable presentation mechanics with no Mixology domain knowledge. Each
toolkit depends on its UI framework and lower-level `pkg` contracts only; architecture checks
prevent imports of `app`, `main`, sibling toolkits, and mismatched surface types.

| Toolkit | Owns | Used by |
| --- | --- | --- |
| [CLI](cli/readme.md) | JSON input/output and reflection-based tables | `main/cli` composition and CLI domain adapters |
| [TUI](tui/readme.md) | Bubble Tea view contracts, list/detail state, layout, components, forms, dialogs, styles, and keys | `main/tui` and TUI domain view models |
| [GUI](gui/readme.md) | Fyne shell, standard pages, async publication, dialogs, semantic controls, tags, and validation | `main/gui` and GUI domain presenters/views |

The relationship is intentionally one-way:

```text
main/<surface>              process lifecycle, routes, top-level composition
        |
app/domains/*/surfaces/*    domain-aware presentation adapters
        |
pkg/toolkits/<surface>      reusable framework mechanics
```

Start with the README for the surface you are changing, then inspect one representative adapter:
ingredients is a compact CRUD example; menus is a useful stateful-workflow example. Keep business
rules in the domain module, route and composition details in `main/<surface>`, and framework
mechanics here only after more than one adapter needs them.
