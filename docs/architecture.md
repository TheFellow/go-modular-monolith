# Architecture

Mixology is one deployable Go application whose bounded contexts own their models, commands,
queries, persistence, Cedar policies, events, and transport adapters. Composition is explicit in
`app.New`; package imports and executable architecture tests enforce the intended boundaries.

## Context map

| Context     | Owns                                                 | Synchronous dependencies              | Events produced                                  |
| ----------- | ---------------------------------------------------- | ------------------------------------- | ------------------------------------------------ |
| Ingredients | ingredient catalog                                   | —                                     | created, updated, deleted                        |
| Drinks      | recipes                                              | Ingredients                           | created, updated, deleted                        |
| Inventory   | stock                                                | Ingredients                           | stock adjusted                                   |
| Menus       | curation and publication                             | Drinks, Ingredients, Inventory        | created, drink added/removed, published, drafted |
| Orders      | order lifecycle                                      | Menus, Drinks, Ingredients, Inventory | placed, completed, cancelled                     |
| Audit       | append-only activities                               | —                                     | —                                                |
| Tagging     | polymorphic associations and authorized tag workflow | domain-owned target loaders           | —                                                |

Synchronous collaboration uses public query contracts. Reactive collaboration consumes another
domain's public event; a command may emit only events owned by its own domain. Taggable domains
depend on a narrow repository port and register loaders with the tagging workflow, so tagging never
reaches into private persistence.

The graph is intentionally reciprocal without creating package cycles. Orders query catalog and
stock contracts while Order events cause Inventory to reserve, consume, or release quantities.
Inventory adjustment events can in turn block or unblock every pending Order whose reservation is
affected. Both event families recalculate published Menu availability. Ingredient retirement fans
out similarly: Drinks enter review rather than disappearing, Menu items become unavailable, and
Inventory removes unusable stock while accepted Order snapshots remain historical truth.

## Package boundaries

A regular context exposes its facade at the package root, read contracts in `models`/`queries`, and
events in `events`. Write logic and DAOs are private under `internal`; handlers consume public peer
events but cannot import peer command implementations. Audit and tagging use smaller explicit
profiles appropriate to their responsibilities.

The shared [store](../pkg/store/README.md) is a required bootstrap dependency. Each context
registers its private bstore models during construction; imports have no database-registration side
effects. Invalid registration fails immediately.

Presentation follows a second set of vertical boundaries documented under
[domain surfaces](../app/domains/readme.md#presentation-surfaces). Reusable framework code lives in
the [toolkits](../pkg/toolkits/readme.md); process and cross-domain composition live in `main`.
Domain action projectors bridge those boundaries: they combine Cedar authorization with durable
domain prerequisites and return framework-neutral control state for GUI, TUI, and future web
adapters. Each concrete view then composes transient state such as dirty forms or requests in
flight without moving business rules into widget code.

## Operation pipelines

The shared [middleware pipeline](../pkg/middleware/README.md) sends commands through logging,
metrics, activity tracking, unit of work, authorization, execution, event dispatch, and audit
recording. Authorization evaluates both the loaded input and resulting state, allowing policies to
constrain transitions. Domain mutation, leaf handlers, and successful audit entry share one
transaction. On failure that transaction rolls back, then the failed attempt is audited separately.

Queries share logging and metrics. A get authorizes its returned Cedar entity. A list authorizes
each result and silently removes permission denials; evaluation/infrastructure failures still fail
the query. Counts are computed after authorization to avoid leaking hidden rows.

Entrypoints configure runtime context, store, and application/session. The application never
retains request identity. CLI operations create fresh context per invocation; persistent TUI/GUI
sessions bind one selected actor while still creating fresh operation contexts.

## Events do not cascade

The generated [domain event dispatcher](../pkg/dispatcher/README.md) connects public events to
their handlers without making bounded contexts depend directly on their consumers. Handlers
receive `*middleware.HandlerContext`, which deliberately has no `AddEvent`. They may query
and mutate their own domain and call `TouchEntity`, but cannot emit another event. This makes every
event fan-out a bounded leaf operation.

When several handlers consume one event, their order must be treated as nondeterministic. The
dispatcher therefore runs every optional `PreparingHandler.Handling` method before it runs any
`Handle` method. During this preparation phase, each handler can read and retain the state as it
existed when the original event was raised. Its later `Handle` call can use that snapshot even if
another handler has since changed related state in the shared transaction. This two-phase protocol
preserves the information that might otherwise require a follow-up, cascading event, without making
correctness depend on handler order.

Order placement demonstrates why preparation and transactional fan-out matter: one event may
touch several inventory rows and menus, and any reservation failure rolls back the Order and every
handler mutation. Handler changes are recorded as audit touches on the initiating operation.

Ingredient retirement is another deliberate fan-out. Ingredients owns validation of an optional
explicit permanent replacement. Drinks owns canonical recipe rewrite or `review_required` state;
Inventory removes unusable stock; Menus preserves published curation while recalculating degraded
availability; Orders blocks historical snapshots rather than rewriting an accepted order. Optional
recipe references and substitute candidates follow less destructive rules than required canonical
references. These are leaf reactions in one transaction and do not emit follow-up events.

Menus owns a computed, authorized readiness report. It is evaluated both as a query and inside the
authoritative Publish command. Findings distinguish blockers (invalid canonical state, unavailable
items, or temporary substitution) from operational warnings such as low stock. This permits an
already-published Menu to represent degradation honestly while preventing a draft from being
promoted into a state the application already knows is unsuitable.

## Authorization

Each domain owns a Cedar schema and policies; the shared
[authorization package](../pkg/authz/README.md) validates and assembles them into one policy set.
Public actors are owner, manager, sommelier, bartender, and anonymous. Gets/commands return typed
permission errors; lists elide denied entities. Taggable entities expose native Cedar string tags,
enabling policy-owned ABAC without giving tags application-global meaning.

Authorization and action availability are deliberately separate. A denied action is omitted from
the presentation; an authorized action whose domain prerequisite is unmet remains visible but
disabled with an explanation. A group's permission is only a default: a control with a distinct
operation, such as Publish, overrides Edit rather than inheriting it. The shared
[action presentation model](../pkg/presentation/actions/README.md) evaluates these declarations, but
commands remain authoritative and repeat authorization and invariants against current state. UI
projection is guidance, not protection against stale state or time-of-check/time-of-use races.

## Generation

| Generator               | Output                                            |
| ----------------------- | ------------------------------------------------- |
| `pkg/dispatcher/gen`    | event-to-handler type-switch wiring               |
| `pkg/authz/gen`         | assembled policies plus domain Cedar models/tests |
| `app/kernel/entity/gen` | strongly typed prefixed entity IDs                |
| `pkg/errors/gen`        | typed constructors and matching test assertions   |

Run `go generate ./...` and commit source and generated output together.

## Enforcement

`.arch-lint.yaml` prevents toolkit-to-application imports, sibling-toolkit coupling, mismatched
surface/toolkit imports, surface-to-composition imports, cross-domain presentation coupling,
shared-package domain imports, private authz/internal access, foreign-event emission, and query or
handler access to command implementations. Tests in `architecture/` also validate every context's
allowed topology and require every domain to be initialized by `app.New`.

Typed errors are transport-neutral: one immutable kind maps to HTTP, gRPC, CLI, and TUI semantics
while separating diagnostic detail from safe presentation text.
