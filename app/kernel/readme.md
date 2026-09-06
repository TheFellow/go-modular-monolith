# Kernel Packages

Kernel packages contain foundational value types shared across domains.

## Guidelines

1. Simple value types only: immutable-ish value objects with validation/formatting.
2. Dependency direction: domains may depend on kernel; kernel must not depend on `app/domains/**`.
3. Keep the surface minimal; domain logic stays in domains.

Measurement values validate supported units and reject nonfinite quantities. `Unit.Canonical()`
maps volume units to `ml` and retains native discrete units; convert an amount rather than changing
its unit label. Inventory owns the choice to persist canonical quantities and expose separate
display and cost units. Money values retain their currency; callers must check currency equality
before calculating a margin.
