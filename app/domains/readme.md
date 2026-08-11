# Bounded contexts

Each directory here is a vertical domain module. Its root `module.go` is the application-facing
facade; public `models`, `queries`, and `events` are deliberate collaboration contracts. Command
implementations and persistence stay under `internal`, authorization definitions stay owned by the
domain, and architecture tests reject undeclared package shapes and dependency directions.

See the [architecture guide](../../docs/architecture.md) for the context map, pipelines, event
rules, and enforcement details.

## Presentation surfaces

`surfaces/cli`, `surfaces/tui`, and `surfaces/gui` adapt a domain's public API to one presentation
framework. They may import their matching [presentation toolkit](../../pkg/toolkits/readme.md), but
not executable composition, sibling domains' concrete presentations, or another surface toolkit.

Responsibilities are split as follows:

| Layer                                     | Responsibility                                                             |
| ----------------------------------------- | -------------------------------------------------------------------------- |
| `main/<surface>`                          | process lifecycle, runtime configuration, routes, cross-domain composition |
| `app/domains/<domain>/surfaces/<surface>` | domain-aware view models/presenters, validation, view conversion, commands |
| `pkg/toolkits/<surface>`                  | reusable framework mechanics with no application knowledge                 |

To add a surface operation, expose it from the module first, implement the domain adapter next,
then wire it at the entrypoint. This keeps every adapter independently testable and prevents the
executable from accumulating domain behavior.

Mutable public models expose an opaque `Revision` so replace-style editors and document adapters
can round-trip the version they read. Surfaces transport that value but do not compare or increment
it; private DAO rows mark it with `store:"revision"`, and the store owns the atomic conflict check.
