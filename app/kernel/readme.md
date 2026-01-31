# Kernel Packages

Kernel packages contain foundational value types shared across domains.

## Guidelines

1. Simple value types only: immutable-ish value objects with validation/formatting.
2. Dependency direction: domains may depend on kernel; kernel must not depend on `app/domains/**`.
3. Keep the surface minimal; domain logic stays in domains.

