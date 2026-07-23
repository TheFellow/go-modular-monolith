# Building High-Quality Software: A Tutorial Series

This codebase is a teaching vehicle. Each lesson isolates a principle and shows how the Mixology
project applies it. The goal is not to explain every line — it's to show that clean architecture
emerges from a small number of simple, enforced rules.

## Lesson 1: Boundaries Are Walls, Not Lines

A boundary you can accidentally cross is not a boundary. This project enforces module isolation
at compile time using `arch-lint`.

Seven rules in `.arch-lint.yaml` define what can import what. They run in CI. If you violate one,
the build breaks.

**The rules, in plain English:**

1. Shared code cannot import domain code
2. Domain A cannot reach into Domain B's internals
3. Event handlers cannot import command implementations
4. Event definitions cannot depend on internal packages
5. Queries cannot import write-side code
6. Models cannot depend on internal packages
7. Handlers must use queries/events/models — never the module root

**Where to look:** `.arch-lint.yaml`, then try adding an illegal import and watch `go tool arch-lint` fail.

**The takeaway:** Don't document rules. Enforce them. If a rule isn't enforced, it will be broken
the day after it's written.

## Lesson 2: Make Illegal States Unrepresentable

The type system should prevent bugs, not catch them.

**Typed IDs.** Every entity has its own ID type (`DrinkID`, `MenuID`, `IngredientID`). You
cannot pass a `DrinkID` where a `MenuID` is expected. These are generated from a single
`Entities` slice via `app/kernel/entity/gen`.

**Exhaustive switches.** Two linters — `exhaustive` and `go-check-sumtype` — ensure every enum
value and sum type variant is handled. Add a new drink category and forget a case? The build
breaks.

**HandlerContext.** Event handlers receive `*middleware.HandlerContext`, not the full
`*middleware.Context`. `HandlerContext` exposes `Transaction()`, `TouchEntity()`, and
`Principal()`. It deliberately **omits** `AddEvent()`. Handlers literally cannot emit new
events — the no-cascading invariant is enforced by the compiler, not by convention.

**Where to look:** `app/kernel/entity/entities.go`, `pkg/middleware/context.go` (search for
`HandlerContext`), then try calling `AddEvent()` from a handler.

**The takeaway:** If a constraint matters, encode it in types. A comment saying "don't do X" is
a bug waiting to happen. A type that makes X impossible is a guarantee.

## Lesson 3: One Pipeline, Every Concern

Cross-cutting concerns (logging, metrics, auth, transactions) should be invisible to domain code.

Every operation enters through a middleware pipeline. Domain code never calls a logger, starts a
transaction, or checks permissions. The pipeline does it.

**Command pipeline:** Logging → Metrics → TrackActivity → UnitOfWork → DispatchEvents.
After `RunCommand` loads the current entity, typed `AuthorizeCommand` middleware checks the
loaded state, invokes the handler, and checks the result (the "dual-state check" — can you touch
the current state AND create the resulting state?).

**Query pipeline:** Logging → Metrics → Execute → result-aware AuthZ. `RunEntityQuery` authorizes
a successfully loaded entity and returns lookup errors unchanged. `RunListQuery` authorizes
each candidate and elides denied entities instead of returning a permission error. Counts use
the visible result set.

**Where to look:** `pkg/middleware/chains.go` for dependency-bound pipeline construction, then
`pkg/middleware/authz.go` for the typed authorization wrappers and `pkg/middleware/run.go` for
query and command execution.

**The takeaway:** If you're writing `log.Info()` or `if !authorized` inside a command handler,
you're in the wrong layer. Push cross-cutting concerns into infrastructure. Domain code should
only contain domain logic.

## Lesson 4: Events Carry State, Not References

When something happens, broadcast what happened — not a pointer to go look it up.

Events in this project are "fat": `DrinkCreated` carries the full `Drink` struct, not just an ID.
Handlers can react to the changed aggregate without reloading it. A handler may still query its
own read model to discover dependent entities, such as the menus containing a deleted drink.

When `IngredientDeleted` fires, both the Drinks handler and the Menus handler receive it
independently. They run in the same transaction. The Drinks handler soft-deletes affected
recipes. The Menus handler removes affected drinks from menus. Neither knows the other exists.
They don't cascade — the dispatcher invokes each leaf handler for the same event, in order,
inside the originating command's unit of work. If any handler fails, all command and handler
writes roll back together.

The dispatcher is generated code. `pkg/dispatcher/gen` scans handler methods via AST and produces
a type-switch that routes every event to its handlers. When multiple handlers implement
`PreparingHandler`, the generated code calls all `Handling()` methods before any `Handle()`,
giving handlers a chance to query data before mutations begin.

**Where to look:** Any event in `app/domains/*/events/`, then `pkg/dispatcher/dispatcher_gen.go`
to see the generated routing, then `app/domains/menus/handlers/ingredient-deleted.go` for a
real handler.

**The takeaway:** Fat events trade a bit of memory for simpler reactions. Dependent-entity
queries remain domain-owned, and the prepare-before-handle phase removes mutation-order
dependencies when several handlers share an event.

## Lesson 5: Generate the Boring Parts

Repetitive wiring code is a maintenance liability. Generate it.

Four generators follow the same pattern: a `gen/` subdirectory with a Go program, invoked by
`//go:generate go run ./gen` in the parent.

| Generator | What it generates |
|-----------|-------------------|
| `pkg/dispatcher/gen` | Event → handler routing (type-switch dispatch) |
| `pkg/authz/gen` | Cedar policy assembly from per-domain `.cedar` files |
| `app/kernel/entity/gen` | Strongly-typed entity IDs with parse/validate/format |
| `pkg/errors/gen` | Error constructors + matching testutil assertions |

**Where to look:** Pick any generator's `gen/` directory, read the ~100-line Go program, then
look at the `_gen.go` file it produces.

**The takeaway:** If you're writing the same structural code across N domains, write it once as a
generator. The generator is the source of truth. The generated code is a build artifact.

## Lesson 6: Errors Are a Contract

An error is not a string. It's a contract between the layer that fails and every surface that
renders it.

Each immutable `Kind` specification in `pkg/errors` maps a single category to four transport codes simultaneously:
HTTP status, gRPC code, CLI exit code, and TUI display style. Domain code returns
`errors.NotFoundf("drink %s", id)`. The CLI renders exit code 20. The TUI renders a styled
warning. A future HTTP API would render 404. The domain never thinks about transport.

The error generator also produces matching assertion helpers in `pkg/testutil`:
`ErrorIsNotFound(t, err)`, `ErrorIsPermission(t, err)`, etc. Generated typed wrappers share a
common error payload that keeps internal diagnostic detail separate from presentation-safe text.

**Where to look:** `pkg/errors/kind.go` for the taxonomy and transport specifications,
`pkg/errors/error.go` for the shared payload, then `pkg/errors/gen` for the generator.

**The takeaway:** Errors should be typed, not stringly-typed. Define the error vocabulary once
and let every surface map it to its own protocol. Never let a CLI exit code or HTTP status
leak into domain logic.

## Lesson 7: Authorization as Policy, Not Code

Permission logic should be data you can read, not code you have to trace.

Authorization uses [Cedar](https://www.cedarpolicy.com/) policies. Each domain defines its
policies in `<domain>/authz/` as `.cedar` files — declarative rules, not Go conditionals. The
`pkg/authz/gen` generator assembles all policies into a single `PolicySet` at startup.

Five roles: `owner` (full), `manager` (operations and full catalog visibility), `sommelier`
(wine drinks plus permitted menu/ingredient operations), `bartender` (non-wine drinks and
orders), and `anonymous` (public read-only views).

Both the input and output of every command must satisfy the `CedarEntity` interface — the system
can evaluate authorization against the entity's actual state, not just its type.

Query results use the same representation. A denied get returns a permission error, while a
denied list item is omitted because partial visibility is expected. For example, a sommelier's
drink list contains wines but elides non-wine drinks.

**Where to look:** Any `authz/` directory inside a domain, then `pkg/authz/authorize.go` for
the evaluation engine, then `pkg/middleware/authz.go` for command, get, and list semantics.

**The takeaway:** When you can read authorization rules in a `.cedar` file without understanding
Go, you've separated policy from mechanism. Anyone can audit who can do what without reading
source code.

## Lesson 8: One Binary, Every Surface

A CLI, a TUI, and (eventually) an API should share 100% of their business logic.

The `main/cli` binary serves both the CLI and the TUI (`--tui` flag). Both surfaces import
domain modules and call the same `RunCommand`, `RunEntityQuery`, and `RunListQuery` functions
through the same middleware pipelines. Authorization, logging, metrics, and audit tracking
apply identically regardless of surface.

Each domain has a `surfaces/` directory with `cli/` and `tui/` subdirectories. These are thin
presentation layers — they map user input to domain calls and format output. They contain zero
business logic.

List filtering follows the same rule. Public list requests carry a human-readable Expr filter,
and domain-owned tagged view structs define the accepted schema. `pkg/filter` type-checks the
expression and converts it into an application-owned tree that can cross CLI, TUI, or future
gRPC boundaries. Safe conjunctive comparisons are pushed into bstore while the full predicate is
retained for exact residual evaluation of parentheses, `or`, `not`, nested fields, and string
operations. Every CLI list command exposes its concrete schema and examples through
`--filter-help`.

**Where to look:** Pick a domain's `surfaces/cli/` and `surfaces/tui/` side by side. Notice they
call the same module methods. Then check `main/cli/cli.go` for the `--tui` flag.

**The takeaway:** If adding a new surface (HTTP, gRPC) requires changing domain code, your
domain is coupled to your transport. Surfaces should be interchangeable shells around the same
core.

## Lesson 9: Test the Pipeline, Not the Plumbing

Tests should exercise the same code paths as production.

`pkg/testutil` provides a `Fixture` that stands up the full middleware pipeline with an isolated
temporary database. Fixture creation opens a wrapping transaction, propagates it through the
application session, and registers a `t.Cleanup` rollback. Tests can run in parallel without
leaking state or paying for teardown deletes. They call the same module methods that the CLI and
TUI call, so logging, authorization, command handling, event dispatch, cross-domain handler
writes, and audit recording all execute together.

Bootstrap helpers create domain entities through public application operations. The drink
builder accepts either an ingredient name or a concrete ingredient model, and `AddDrinks`
attaches any number of drinks to a menu without repeating command plumbing:

```go
f := testutil.NewFixture(t)
b := f.Bootstrap()
lime := b.WithIngredient("Fresh Lime", measurement.UnitOz)
drink := f.CreateDrink("Daiquiri").WithIngredient(lime, 1).Build()
menu := b.AddDrinks(b.WithMenu("Classics"), drink)
```

For cross-domain event tests, `LatestAuditEntry` finds the originating activity and
`AuditTouches` compares the complete touched-entity set, making accidental mutations and missing
audit attribution visible.

`ActorContext()` switches between principals so authorization tests are trivial:

```go
ctx := f.OwnerContext()          // full access
ctx  = f.ActorContext("bartender") // restricted
_, err := f.Drinks.Delete(ctx, drinkID)
testutil.ErrorIsPermission(t, err)
```

**Where to look:** Any `_test.go` file in `app/domains/*/`. Then `pkg/testutil/fixture.go`.

**The takeaway:** If your tests mock the database, skip auth, and bypass middleware, they're
testing a system that doesn't exist. Test the real stack. Keep the real stack fast enough to
test.

## Lesson 10: The Simplest Thing That Works

This is a modular monolith. One binary. One database. No message broker. No service mesh. No
container orchestration.

It uses bstore (embedded bbolt) — zero external dependencies. ACID transactions.
`UnitOfWork` middleware means if a handler fails, everything rolls back. The database is a file.
The store is injected during app bootstrap, and each bounded context registers its internal
persistence model while it is constructed. Invalid bootstrap wiring panics immediately, and
package imports do not mutate a global registry.

The project could add gRPC, Kafka, and Kubernetes. It doesn't need to yet. Every piece of
infrastructure in this codebase exists because it solves a problem that actually occurred, not
because it might occur.

**The takeaway:** Complexity is not a feature. Start with the simplest architecture that
enforces your invariants. Add infrastructure when you have evidence you need it, not when you
imagine you might.
