# Presentation toolkits

`pkg/toolkits` contains reusable presentation mechanics with no Mixology domain knowledge. Each
toolkit depends on its UI framework and lower-level `pkg` contracts only; architecture checks
prevent imports of `app`, `main`, sibling toolkits, and mismatched surface types.

| Toolkit                      | Owns                                                                                               | Used by                                        |
| ---------------------------- | -------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| [Actions](actions/readme.md) | Framework-neutral authorization and availability projection                                        | GUI, TUI, and future web domain adapters       |
| [CLI](cli/readme.md)         | JSON input/output and reflection-based tables                                                      | `main/cli` composition and CLI domain adapters |
| [TUI](tui/readme.md)         | Bubble Tea view contracts, list/detail state, layout, components, forms, dialogs, styles, and keys | `main/tui` and TUI domain view models          |
| [GUI](gui/readme.md)         | Fyne shell, standard pages, async publication, dialogs, semantic controls, tags, and validation    | `main/gui` and GUI domain presenters/views     |

The relationship is intentionally one-way:

```text
main/<surface>              process lifecycle, routes, top-level composition
        |
app/domains/*/surfaces/*    domain-aware presentation adapters
        |
pkg/toolkits/<surface>      reusable framework mechanics
```

`pkg/toolkits/actions` sits beside the framework toolkits rather than beneath one of them. A
domain-owned projector declares stable control IDs, permission inheritance or overrides, and
business conditions once. Each surface maps the resulting visible/enabled states into native
widgets or key bindings and adds only its transient interaction constraints.

Start with the README for the surface you are changing, then inspect one representative adapter:
ingredients is a compact CRUD example; menus is a useful stateful-workflow example. Keep business
rules in the domain module, route and composition details in `main/<surface>`, and framework
mechanics here only after more than one adapter needs them.

## Why the package shapes differ

Package symmetry is useful when it reflects the same ownership boundary, but the UI runtimes do
not produce identical seams. TUI forms and dialogs are autonomous Bubble Tea models with their own
state, messages, keys, styles, and tests, so they are useful subpackages. CLI tables are likewise an
independent renderer. The GUI shell, standard pages, semantic controls, tables, tags, dialogs, and
async publication types are currently consumed together and share framework-native contracts;
splitting them would add sibling imports without creating independent ownership.

Extract a GUI subpackage when a cluster gains a narrow API, can avoid importing the GUI toolkit
root, and has a useful consumer or lifecycle of its own. Do not add matching directories solely to
make the trees look alike.
