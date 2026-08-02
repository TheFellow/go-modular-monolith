# Architecture

Mixology is one deployable Go application whose bounded contexts own their models, commands,
queries, persistence, Cedar policies, events, and transport adapters. Composition is explicit in
`app.New`; package imports and executable architecture tests enforce the intended boundaries.

## Context map

| Context | Owns | Synchronous dependencies | Events produced |
| --- | --- | --- | --- |
| Ingredients | ingredient catalog | — | created, updated, deleted |
| Drinks | recipes | Ingredients | created, updated, deleted |
| Inventory | stock | Ingredients | stock adjusted |
| Menus | curation and publication | Drinks, Ingredients, Inventory | created, drink added/removed, published, drafted |
| Orders | order lifecycle | Menus, Drinks, Ingredients, Inventory | placed, completed, cancelled |
| Audit | append-only activities | — | — |
| Tagging | polymorphic associations and authorized tag workflow | domain-owned target loaders | — |

Synchronous collaboration uses public query contracts. Reactive collaboration consumes another
domain's public event; a command may emit only events owned by its own domain. Taggable domains
depend on a narrow repository port and register loaders with the tagging workflow, so tagging never
reaches into private persistence.

## Package boundaries

A regular context exposes its facade at the package root, read contracts in `models`/`queries`, and
events in `events`. Write logic and DAOs are private under `internal`; handlers consume public peer
events but cannot import peer command implementations. Audit and tagging use smaller explicit
profiles appropriate to their responsibilities.

The store is a required bootstrap dependency. Each context registers its private bstore models
during construction; imports have no database-registration side effects. Invalid registration
fails immediately.

Presentation follows a second set of vertical boundaries documented under
[domain surfaces](../app/domains/readme.md#presentation-surfaces). Reusable framework code lives in
the [toolkits](../pkg/toolkits/readme.md); process and cross-domain composition live in `main`.

## Operation pipelines

Commands pass through logging, metrics, activity tracking, unit of work, authorization, execution,
event dispatch, and audit recording. Authorization evaluates both the loaded input and resulting
state, allowing policies to constrain transitions. Domain mutation, leaf handlers, and successful
audit entry share one transaction. On failure that transaction rolls back, then the failed attempt
is audited separately.

Queries share logging and metrics. A get authorizes its returned Cedar entity. A list authorizes
each result and silently removes permission denials; evaluation/infrastructure failures still fail
the query. Counts are computed after authorization to avoid leaking hidden rows.

Entrypoints configure runtime context, store, and application/session. The application never
retains request identity. CLI operations create fresh context per invocation; persistent TUI/GUI
sessions bind one selected actor while still creating fresh operation contexts.

## Events do not cascade

Handlers receive `*middleware.HandlerContext`, which deliberately has no `AddEvent`. They may query
and mutate their own domain and call `TouchEntity`, but cannot emit another event. This makes every
event fan-out a bounded leaf operation.

When several handlers consume one event, their order must be treated as nondeterministic. The
dispatcher therefore runs every optional `PreparingHandler.Handling` method before it runs any
`Handle` method. During this preparation phase, each handler can read and retain the state as it
existed when the original event was raised. Its later `Handle` call can use that snapshot even if
another handler has since changed related state in the shared transaction. This two-phase protocol
preserves the information that might otherwise require a follow-up, cascading event, without making
correctness depend on handler order.

## Authorization

Each domain owns a Cedar schema and policies; generation assembles them into one policy set. Public
actors are owner, manager, sommelier, bartender, and anonymous. Gets/commands return typed permission
errors; lists elide denied entities. Taggable entities expose native Cedar string tags, enabling
policy-owned ABAC without giving tags application-global meaning.

## Generation

| Generator | Output |
| --- | --- |
| `pkg/dispatcher/gen` | event-to-handler type-switch wiring |
| `pkg/authz/gen` | assembled policies plus domain Cedar models/tests |
| `app/kernel/entity/gen` | strongly typed prefixed entity IDs |
| `pkg/errors/gen` | typed constructors and matching test assertions |

Run `go generate ./...` and commit source and generated output together.

## Enforcement

`.arch-lint.yaml` prevents toolkit-to-application imports, sibling-toolkit coupling, mismatched
surface/toolkit imports, surface-to-composition imports, cross-domain presentation coupling,
shared-package domain imports, private authz/internal access, foreign-event emission, and query or
handler access to command implementations. Tests in `architecture/` also validate every context's
allowed topology and require every domain to be initialized by `app.New`.

Typed errors are transport-neutral: one immutable kind maps to HTTP, gRPC, CLI, and TUI semantics
while separating diagnostic detail from safe presentation text.
