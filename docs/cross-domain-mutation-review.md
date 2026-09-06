# Cross-domain mutation review

Reviewed 2026-09-06 against `bc52eb8`. The findings below preserve the original baseline review. User decisions and implemented behavior are recorded in [Transactional domain workflows](transactional-workflows.md). In particular, the original extra-phase suggestion in #4 was rejected: implementation uses the existing stateful `Handling`/`Handle` protocol. Schema changes use fresh seed data with no migration. Evidence links are pinned to the reviewed commit so they remain accurate after the implementation changes below.

## Baseline findings (superseded by the workflow record)

The transaction foundation is sound: command writes, leaf reactions, and successful activity records share one SQLite transaction. The main gaps are incomplete dependency discovery, reactions that depend on sibling handler order, mutable catalog data being used to render historical orders, and lifecycle policy that differs between ingredients, drinks, menus, and stock.

Permanent ingredient replacement already exists. It validates category and unit compatibility, rewrites recipes, removes old stock, updates menu availability, blocks affected open orders, and records participants in one transaction. It deliberately does **not** replace an accepted order's ingredient snapshot. Extending that behavior requires an explicit order-amendment policy.

Complexity: **S** = localized change and integration tests; **M** = multiple domains/contracts or surface changes; **L** = lifecycle/versioning changes, migration, and workflow tests. These are relative engineering sizes, not elapsed-time estimates. “Fix” means no new product policy is needed; “Decision” identifies a product choice; “Mixed” contains both.

| Priority | Finding | Impact | Complexity | Classification |
| --- | --- | --- | --- | --- |
| 1 | Client input can bypass lifecycle operations and strand reservations | Critical | S–M | Fix |
| 2 | Ingredient unit changes corrupt reservation interpretation and cost basis | Critical | M | Mixed |
| 3 | Reservation release does not restore other blocked orders | High | S–M | Fix |
| 4 | Transactional fan-out relies on sibling handler order | High | M | Fix / architectural choice |
| 5 | Implicit substitutions are absent from dependency tracking and use mutable names | High | M | Mixed |
| 6 | Historical orders still depend on live menu/drink records | High | L | Decision |
| 7 | No atomic amendment/replacement workflow for in-flight orders | High | L | Decision |
| 8 | Ingredient retirement conflates discontinuation with stock disposal | High | M–L | Decision |
| 9 | Drink deletion and menu lifecycle mutate or hide published history | High | M–L | Decision |
| 10 | Audit captures participants, but cannot explain/reconstruct changes; composed failures disappear | High | M–L | Mixed |
| 11 | Recipe/catalog validation permits incompatible cross-domain state | High | M | Fix |
| 12 | Tolerant availability calculations are used for persisted mutations | Medium–High | M | Fix |
| 13 | Cost calculations can choose a different, impossible fulfillment plan | Medium | S–M | Fix |
| 14 | Inventory sets and full tag replacements lack client concurrency preconditions | Medium–High | M | Mixed |
| 15 | Shortage handling blocks every order rather than allocating remaining stock | Medium | M–L | Decision |
| 16 | Optional ingredients are omitted from all reservation/consumption accounting | Medium | M | Decision |

The following map covers the baseline mutation families across all seven contexts. A missing event alone is not a defect; reactions are needed only where an invariant or deliberate projection depends on the mutation.

| Origin / mutation | Current recipients or dependency | Assessment |
| --- | --- | --- |
| Ingredients create | Catalog; audit; optional composed tags | Normally no downstream reaction needed; creation can nevertheless enable a name-based implicit substitution (#5). |
| Ingredients update | Drinks/Menus are touched in audit; their handlers do not rewrite state | Unit, identity, and compatibility changes are not handled (#2, #5, #11). |
| Ingredients retire/replace | Drinks, Inventory, Menus, Orders, Audit; inventory tags removed | Existing atomic fan-out; ordering, stock disposition, and order recovery need work (#4, #7, #8, #10). |
| Drinks create/update | Ingredients validation; Menu availability on update; Audit | Incomplete validation and lifecycle input protection; accepted recipes are not versioned (#1, #6, #9, #11). |
| Drinks delete | Removes matching menu items, including published ones; Audit | No Orders reaction. Accepted usage survives, but order presentation still requires the drink (#6, #9). |
| Inventory set/adjust | Orders shortage state; Menu availability; Audit | Missing implicit dependencies and unsafe unit conversion; cost-only changes are read live (#2, #5, #13, #14). |
| Orders place | Menu/Drink/catalog/stock queries; Inventory reservations; Menu availability; Audit | Atomic stock reservation is present. Snapshot and input ownership are incomplete (#1, #6, #16). |
| Orders complete | Inventory consumption and reservation deletion; Menu availability; Audit | Terminal command is idempotent. Add reservation consistency checks (#3); retain historical snapshots (#6). |
| Orders cancel | Inventory release; Menu availability; Audit | Other blocked Orders are not reconsidered (#3). Cancellation time/reason is not a first-class record (#10). |
| Menus create/rename/add/remove | Draft rules, Drink existence/availability, Audit | Existing direct edit guards are useful. Item edits do not amend accepted Orders; historical display should not depend on them (#6). |
| Menus publish/draft/delete | Readiness, availability refresh, Audit | Drafting stops new placement; accepted Orders continue. Delete is draft-only, but drafting a used menu then deleting it hides historical context (#6, #9). |
| Tagging upsert/remove/replace | Domain-owned target loaders and permissions; Audit | No inherent stock/recipe effect is required: tags are policy-owned metadata. Full-set concurrency and historical tag evidence are incomplete (#10, #14). |
| Audit append | Shared transaction for successful writes | Append-only activity is present. No reverse business mutation is needed (#10). |

1. **Protect lifecycle-owned fields at command boundaries — Fix, S–M.**

   `Drinks.Update` copies the submitted model, including `DeletedAt`. An ordinary update can therefore retire a drink without `DrinkDeleted` or its menu-removal behavior. `Orders.Place` likewise copies `DeletedAt` and `BlockedIngredients` from caller input. A placed order with `DeletedAt` set reserves stock but cannot subsequently be fetched or cancelled through the normal facade.

   **Evidence:** [drink update](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/drinks/internal/commands/update.go), [order placement](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/orders/internal/commands/place.go). Both deletion-field cases were reproduced through the real application pipeline.

   **Implement:** separate command input from persisted entity state, or explicitly allowlist writable fields; initialize lifecycle state inside the domain. Apply the same audit to create/update contracts across the catalog. This preserves the existing dedicated lifecycle permissions and events.

   **Tests:** submit deletion timestamps, block lists, terminal timestamps, and status fields through ordinary create/update; assert rejection or safe normalization, no hidden live reservations, and the expected event/audit behavior. No product decision needed.

2. **Make ingredient units safe across Inventory, Orders, recipes, and costs — Mixed, M.**

   Ingredient update accepts an arbitrary nonempty unit. Inventory does not consume `IngredientUpdated`. A later stock adjustment converts stock to the new catalog unit but reservation sums simply add stored numeric quantities and label the result with the current stock unit. A reproduced example changes 10 oz of stock to 295.74 ml while a 2 oz reservation becomes reported as **2 ml**. `Adjust` also retains the old numeric cost per unit unless a replacement cost is supplied, potentially changing the implied cost basis. Incompatible dimensional changes can make later fulfillment fail.

   **Evidence:** [ingredient update](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/ingredients/internal/commands/update.go), [inventory adjustment](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/inventory/internal/commands/adjust.go), [reservation sums](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/inventory/internal/dao/reservations.go), [inventory hydration](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/inventory/internal/dao/get.go).

   **Fix regardless of policy:** validate supported units and convert reservation quantities before aggregation. Never silently relabel quantities or prices. Detect already-mixed rows during migration.

   **Options:** **A — freeze unit after usage (recommended initially):** simple, stable identity; requires a new ingredient for a substantive change. **B — explicit compatible-unit conversion command:** convenient display/storage changes, but must atomically convert stock, reservation rows, cost basis, and relevant projections, while preserving historical snapshots. **C — canonical storage units plus display units:** strongest long-term separation; broadest migration. Reject dimensional changes on used ingredients under all three.

   **Tests:** oz↔ml with open orders, completion after conversion, mixed reservation units, unchanged economic cost, incompatible dimensions, rollback of every participant.

3. **Reconcile other Orders after reservation release — Fix, S–M.**

   With two orders reserving 2 oz each and corrected stock of 2 oz, both orders become blocked. Cancelling one leaves the other blocked even though its reservation is now fully backed. Orders reacts to `StockAdjusted` and ingredient retirement, but not cancellation. Requiring a fake stock adjustment to repair this is a missing reaction, not a product choice.

   **Evidence:** [Order handlers](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/orders/handlers/stock-adjusted.go), [reservation release](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/inventory/handlers/order-cancelled.go). Reproduced with real commands.

   **Implement:** reconsider all affected open orders after the reservation change, preserving independent blockers such as another shortage or ingredient retirement. Use the coordination approach in #4. Touch the recovered orders in the cancellation activity.

   **Additional hardening:** completion currently succeeds if no reservations exist and clamps negative consumption to zero. For a nonempty usage snapshot, verify reservation presence, quantities, units, and backing stock; reject inconsistencies rather than reporting successful fulfillment. Empty usage for an optional-only order is a separate policy (#16).

   **Tests:** cancellation repairs peers, multiple blockers, cancellation after retirement, repeated cancellation, invalid reservation state, and failure rollback.

4. **Remove accidental handler-order dependence — Fix / architectural choice, M.**

   Menu order handlers read Inventory after its sibling handler has changed reservations. Menu retirement reads Drinks/Inventory after their sibling handlers have rewritten/deleted data. Preparation snapshots identify affected entities but do not make those final-state reads independent of `Handle` ordering.

   **Evidence:** [generated dispatcher](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/pkg/dispatcher/dispatcher_gen.go), [menu reservation reaction](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/menus/handlers/order-placed.go), [menu retirement reaction](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/menus/handlers/ingredient-deleted.go). Running the same placement handlers in opposite orders produces `unavailable` versus `available` for a menu whose stock has all been reserved. Current generated order masks the defect.

   **Recommendation:** use explicit bounded phases inside the same transaction: prepare the affected set, apply domain-owned effects, then refresh dependent projections against final state. Retain leaf handlers and prohibit event cascades; require order independence within each phase. An alternative is a complete prepared effect plan consumed by independent handlers, but it is more complex and risks duplicating domain logic. Sorting handlers by name or hardcoding current order is not sufficient.

   **Tests:** permute handlers within the supported phases for placement, cancellation, completion, and replacement; assert identical persisted state and touches. Inject failure in the last phase and prove all writes roll back. This is a technical architecture choice, with no new user-facing policy required.

5. **Track implicit substitutions and give them stable identity — Mixed, M.**

   Fulfillment uses both recipe-declared substitutes and default catalog rules. Dependency discovery (`ListByIngredient`, `drinkUsesIngredient`) sees only explicit recipe references. Exhausting Lemon Juice after Lime Juice is already exhausted leaves a lime drink's stored menu availability `limited`, while a fresh calculation returns `unavailable`. Retirement of an implicit substitute has the same discovery gap.

   Rules for generated ingredient IDs are resolved from normalized **names**. Renaming or creating an ingredient can therefore change fulfillment behavior without a recipe edit. Normalized-name collisions can resolve to an unintended ingredient. Retiring the original also prevents name-based rule lookup through its normal `Get`, unlike explicit substitute IDs retained in a recipe.

   **Evidence:** [rule resolution](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/ingredients/queries/substitutions.go), [stock dependency check](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/menus/handlers/stock-adjusted.go), [recipe dependency query](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/drinks/internal/dao/list.go). Stale stored availability was reproduced.

   **Fix:** expose a single complete dependency contract, including implicit candidates; invalidate all affected menu projections on stock, catalog, and rule changes. Record before/after dependency identity when necessary. Recompute affected draft displays too, or make draft availability explicitly read-time data; current refresh rules differ between retirement and other changes.

   **Options:** **A — persisted ID-based substitution rules (recommended):** stable through renames and auditable; requires rule commands, lifecycle, and migration. **B — recipe-local explicit substitutes only:** simpler ownership and approval; loses shared catalog rules. **C — immutable catalog keys:** smaller change while retaining defaults; still needs complete dependency tracking and collision validation.

   **Tests:** implicit target receipt/depletion/retirement, original and substitute rename, normalized-name collisions, explicit and implicit paths agreeing, and rollback.

6. **Preserve enough order history to stand independently of current catalogs — Decision, L.**

   Orders snapshot aggregate actual ingredient usage and names, which is valuable. They do not capture the accepted drink name, recipe/steps, per-item ingredient allocation, original-to-substitute mapping, menu identity/version/name, or agreed price. The TUI order detail loads the current Menu and Drinks; deleting either breaks historical presentation. Renaming a drink changes how an old order is shown. The displayed total is derived from current menu items, not accepted order data. Menu prices are optional and normal AddDrink currently supplies none, so this is a structural gap even before a full pricing workflow exists.

   **Evidence:** [Order model](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/orders/models/order.go), [snapshot builder](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/orders/internal/commands/complete.go), [order detail](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/orders/surfaces/tui/detail_vm.go). Soft-deleted catalog rows remain in storage, but ordinary facade reads exclude them; retained rows alone do not preserve prior versions.

   **Options:** **A — immutable order acceptance snapshots (recommended minimum):** self-contained history and simple reads; duplicates relevant descriptive data. **B — immutable recipe and menu publication versions referenced by Orders:** stronger provenance and less repeated recipe data; larger model and migration. **C — historical lookup of soft-deleted current rows:** quick recovery of names; cannot recover earlier edits, prices, or accepted recipe steps and is insufficient by itself.

   **Tests:** open, completed, and cancelled orders remain readable after rename, recipe edit, drink retirement, menu redraft/removal/deletion, and ingredient retirement. Verify accepted values remain unchanged and historical reads retain appropriate authorization.

7. **Add an explicit atomic amendment workflow for in-flight replacement — Decision, L.**

   Existing retirement intentionally blocks open orders that used the old ingredient. They can be cancelled; there is no supported route to approve replacement and re-reserve stock for the same order. Silently rewriting `IngredientUsage` would discard what was originally accepted.

   **Evidence:** [ingredient retirement contract](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/ingredients/models/ingredient.go), [order retirement handler](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/orders/handlers/ingredient-deleted.go), [order completion/cancellation](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/orders/internal/commands/complete.go).

   **Options:** **A — cancel and place a linked replacement order:** simplest immutable record; more staff work and no existing atomic linked command. **B — explicit order amendment/revision (recommended):** preserves original acceptance and approved new fulfillment, with atomic release/reserve and actor/reason; requires amendment history and customer/staff approval semantics. **C — bulk automatic amendment during retirement:** lowest operational friction; may change accepted products without explicit order-level approval and makes batch stock allocation more complex.

   **Decisions needed:** may affected orders be amended automatically or only explicitly; must all selected orders succeed or may the operator select a fulfillable subset; does the agreed price stay fixed? Recommended initial behavior is explicit selected-order amendment, all-or-nothing for that selection, unchanged agreed price, and terminal orders excluded. Include review of changed preparation instructions, not just ratios.

   **Tests:** one insufficient replacement rolls back the selection, replacement already used elsewhere in the recipe, ratio conversion and aggregation, repeated request/idempotency, completed/cancelled exclusion, unchanged original acceptance, and complete audit linkage.

8. **Separate ingredient discontinuation from stock withdrawal/disposal — Decision, M–L.**

   Retirement hard-deletes the stock row and its tags, while reservations remain until cancellation. This loses current quantity/cost/tag evidence and makes every existing reservation unusable even if the physical stock could still be served. Replacement does not transfer stock, appropriately: a different ingredient is not automatically the same physical inventory.

   **Evidence:** [retirement stock handler](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/inventory/handlers/ingredient-deleted.go), [stock deletion](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/inventory/internal/dao/delete-by-ingredient.go).

   **Options:** **A — discontinue future use but honor accepted reservations:** preserves usable stock and service; requires separate eligibility for new versus already accepted usage. **B — withdraw/quarantine immediately:** blocks fulfillment and retains stock identity plus reason/quantity; requires an explicit disposal or release operation. **C — reject retirement while stock/reservations exist:** easiest invariant; more manual steps and a poor fit for an urgent withdrawal. Recommended product model is distinct discontinuation and immediate-withdrawal operations, backed by retained stock records and explicit movements.

   **Tests:** usable residual stock, outstanding reservations, quarantine, disposal, terminal order history, stock tags, and no implicit physical transfer to a replacement ingredient.

9. **Define drink withdrawal and menu publication history consistently — Decision, M–L.**

   Ingredient retirement preserves menu curation and degrades availability. Drink deletion instead removes items from every active menu, including published menus, potentially leaving a published menu empty. Direct menu edits are draft-only, so the indirect path bypasses the curation rule. Menu Draft clears `PublishedAt`; a previously used menu can then be edited or deleted. There is no first-class readable archive workflow, despite an `archived` status. A valid Drink update also changes what an already-published menu offers without another publication approval.

   **Evidence:** [drink deletion reaction](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/menus/handlers/drink-deleted.go), [menu draft](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/menus/internal/commands/draft.go), [menu delete](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/menus/internal/commands/delete.go), [drink update reaction](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/menus/handlers/drink-updated.go).

   **Options:** **A — reject withdrawal/edit while referenced by published menus or open orders:** simple and explicit; operationally restrictive. **B — retain a withdrawn menu item, allow accepted orders to finish, require explicit redraft to alter curation (recommended minimum):** consistent with ingredient degradation; requires tombstone/historical display (#6). **C — immutable publications and versioned recipes:** best historical and approval semantics; largest change. With C, draft a new revision while the previous publication remains readable, and separate “stop new orders” from editing.

   Decide separately whether drink withdrawal means “stop selling” or “unsafe to fulfill”; the latter needs explicit order blocking/cancellation, rather than inferring it from deletion. Preserve terminal orders under every option.

   **Tests:** delete the only published item, orders before/after withdrawal, menu redraft/delete with history, recipe changes to published items, and publication timestamps/versions.

10. **Make audit explain the business change and survive composite failures — Mixed, M–L.**

   Audit currently records action, resource, principal, times, success/error, and touched IDs. It cannot explain replacement direction/ratio, which recipe quantities changed, old/new order plans, stock written off, or old/new tags. Touches mix “changed” with “affected/read”: ingredient updates touch dependent Drinks/Menus without changing their rows, while a replacement can change menu meaning without changing its availability enum and therefore without touching the Menu. “Touched” is not a reconstructable change log.

   There is also a concrete transactional coverage gap: `RunTaggedMutation` uses an outer transaction. A tag-step failure rolls back both the domain success activity and failure activity, leaving no record of the rejected composite operation. The existing rollback test explicitly asserts unchanged audit count. Standalone command failure auditing is stronger than composed-workflow failure auditing. A failure after successful audit preparation/at commit is another path to inspect: `TrackActivity` skips a second record once completion is set.

   **Evidence:** [AuditEntry](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/audit/models/entry.go), [activity middleware](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/pkg/middleware/track_activity.go), [tagged composition](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/tagged_mutation.go), [rollback test](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/tagged_mutation_test.go), [menu retirement touch condition](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/menus/handlers/ingredient-deleted.go).

   **Fix:** let the outer workflow own post-rollback failure recording; distinguish attempted participants from committed effects; correlate child activities to one workflow. Preserve atomic success auditing. Capture cancellation time and reason, retirement/replacement reason and identity, and reservation/disposal facts as domain data where operationally needed.

   **Options:** **A — typed effect summaries with before/after values and correlation (recommended):** explains cross-domain decisions without replacing domain storage. **B — full entity snapshots/diffs on every write:** broad forensic detail; larger storage and schema evolution burden. **C — event sourcing:** reconstructable history if designed correctly; substantial architecture change, unnecessary for this teaching goal. Decide whether pure dependency reads belong in a separate participant list instead of the changed-entity list.

   **Tests:** tag authorization failure after domain success, failure in last cross-domain reaction, audit-write/commit failure behavior, one correlated workflow, exact changed and referenced entities, and preserved successful history after soft deletion.

11. **Validate semantic compatibility at write boundaries — Fix, M.**

   Drink create/update checks referenced ingredient existence but not recipe amount compatibility with catalog units. Explicit substitutes are checked for existence, not dimensional compatibility. A stocked incompatible candidate can make strict planning fail even when a valid original is available, because candidate enumeration evaluates it too. Optional ingredient lookup currently suppresses any error and skips its substitute validation. Ingredient Create only checks nonempty category/unit, while Update validates category but still accepts any nonempty unit. Retirement removes optional references without validating the resulting recipe; removing the final optional ingredient can leave a recipe that violates `Recipe.Validate`'s nonempty rule.

   **Evidence:** [drink create](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/drinks/internal/commands/create.go), [drink update](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/drinks/internal/commands/update.go), [recipe validation](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/drinks/models/recipe.go), [retirement rewrite](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/drinks/handlers/ingredient-deleted.go), [candidate planning](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/menus/internal/availability/calculator.go).

   **Implement:** enforce supported units and amount dimensional compatibility; finite positive quantities/ratios at public boundaries; validate substituted and rewritten recipes; distinguish a missing optional ingredient from infrastructure failure. Preserve coherent unavailable/review state if retirement necessarily leaves an invalid canonical recipe, rather than pretending it is active and fulfillable. Category equivalence for *temporary* substitutions is a product rule and should not be invented as part of this fix; physical compatibility is mandatory.

   **Tests:** incompatible optional and required references, incompatible stocked substitute with a valid original, invalid units/categories through all command surfaces, nonfinite amounts, duplicate replacement IDs/amounts, and retirement of the final optional component.

12. **Use strict calculations for authoritative mutations — Fix, M.**

   `Calculate` turns dependency errors into `unavailable`. `CalculateDetail` uses `PickIngredients`, which suppresses planning errors; Menu Publish uses the resulting readiness report. Menu event handlers also persist the tolerant result, and `StockAdjusted.drinkUsesIngredient` treats any drink read error as “not a dependency.” Thus some failures become misleading business blockers, persisted degradation, or skipped updates instead of aborting the transaction. The strict order planning path already demonstrates the intended alternative.

   **Evidence:** [availability calculator](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/menus/internal/availability/calculator.go), [menu publish](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/menus/internal/commands/publish.go), [stock handler](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/menus/handlers/stock-adjusted.go).

   **Implement:** separate presentation fallback from authoritative readiness/projection calculation. Missing or retired stock is a modeled business condition; database/conversion/cancellation errors propagate. Recompute multi-domain reports under a consistent read snapshot when they must describe one state; authoritative commands already have the write transaction.

   **Tests:** injected read failure at every dependency, invalid units, cancellation, unchanged menu state on error, and no success activity for failed reactions. No product decision needed.

13. **Use one complete fulfillment plan for cost and availability — Fix, S–M.**

   Availability and order placement plan the complete requirement set with backtracking. Cost calculation calls `PickIngredient` independently for every line. Two requirements can therefore both cost the same scarce substitute even though only one can consume it and the other must use a more expensive fallback. Availability, suggested price, and actual accepted usage can disagree. Menu margin calculation also subtracts raw decimal amounts without first checking that menu and ingredient-cost currencies match.

   **Evidence:** [cost calculator](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/menus/queries/cost.go), [menu margin](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/menus/queries/analytics.go), [existing shared-substitute fulfillment tests](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/orders/fulfillment_test.go).

   **Implement:** cost the full plan used for the calculation; expose whether it is a current estimate or an accepted-order snapshot (#6). Reject or mark unknown cross-currency margin instead of dividing unlike currencies. Cost-only stock edits need no new availability event if availability truly does not change; estimates should simply read the updated cost.

   **Tests:** shared substitute plus differently priced fallback, repeated canonical ingredient after replacement, mixed currencies, and cost/availability/accepted-plan agreement for the same transaction snapshot.

14. **Extend optimistic concurrency to aggregate editing workflows — Mixed, M.**

   Domain row revisions protect many edits, but Inventory `Set` and `Adjust` inputs have no expected revision. They load the latest stock inside the command, so a stale absolute stock-count form can overwrite intervening completion/receipt changes. Full tag replacement similarly reloads current tag rows; an editor can delete a newly added tag without conflict. Tag mutations do not advance the owning entity's revision, so the domain revision alone cannot protect a combined entity-and-tags edit. SQLite serialization prevents simultaneous writes, but not stale user intent.

   **Evidence:** [inventory input](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/inventory/models/update.go), [stock set](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/inventory/internal/commands/set.go), [tag replacement](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/tagging/module.go), [tag repository](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/tagging/repository.go).

   **Recommendation:** require expected revision for absolute stock edits and an expected tag-set revision/hash for full replacements. Keep additive stock adjustments and explicit one-key tag operations applicable to current state when their semantics allow it. For combined edits, check every precondition before any effect and roll back as one operation.

   **Choice:** guarded editing (recommended, consistent with README promises) versus an explicit privileged unconditional “force recount/replace” command. The latter is useful operationally but must be visibly intentional and audited.

   **Tests:** two connections, stock completion between load/save, concurrent tag addition/removal, unchanged tags accompanying a stale entity editor, and atomic conflict rollback.

15. **Choose how stock shortages allocate existing commitments — Decision, M–L.**

   A stock correction below total reservations blocks every open order using that ingredient, even if available physical stock could satisfy some. This is consistent with the current aggregate shortage rule; unlike #3, changing it is a product choice. No replacement planning is attempted for already accepted orders.

   **Evidence:** [stock shortage event](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/inventory/internal/commands/set.go), [order blocker reaction](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/orders/handlers/stock-adjusted.go).

   **Options:** **A — block all until the aggregate deficit is resolved (recommended initial policy):** simple and avoids choosing winners; stops some fulfillable service. **B — deterministic FIFO/priority allocation:** maximizes fulfillment; requires stable allocation rules and re-evaluation across ingredients. **C — staff-directed allocation/amendment:** explicit business judgment; more workflow and audit requirements. All require #3's recovery behavior when commitments are released.

   **Tests:** multiple orders, partial replenishment, multiple scarce ingredients, deterministic allocation, cancellation recovery, and no overcommit under competing transactions.

16. **Decide what optional ingredients mean for actual consumption — Decision, M.**

   Optional recipe ingredients are skipped by readiness requirements, order snapshots, reservation, and cost calculation. If a garnish is physically used, the system never consumes it. An optional-only recipe can place/complete an order with no stock usage. “Not required for availability” and “never consumed” are currently the same concept.

   **Evidence:** [order requirements](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/orders/internal/commands/complete.go), [availability requirements](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/menus/internal/availability/calculator.go), [cost calculation](https://github.com/TheFellow/go-modular-monolith/blob/bc52eb81ad35c8f1a3863312aefc1f574ee46493/app/domains/menus/queries/cost.go).

   **Options:** **A — optional means descriptive/untracked (smallest scope):** document the limitation and explicitly decide whether optional-only recipes are permitted. **B — include when available:** closer to service behavior; must snapshot each chosen optional and reserve it atomically. **C — explicit per-order inclusion/omission (recommended if optional items matter):** accurate preparation and accounting; requires item-level fulfillment plans and UI choices. Acceptance snapshots should retain what was actually selected under B/C.

   **Tests:** optional depletion does not block required service, included optional stock is consumed exactly once, omitted garnish costs nothing, retirement handling, and optional-only recipes.

Implementation should start with #1 and the policy-independent safety portions of #2/#3/#5/#11–#14. Settle #4 before adding more fan-out. Decide history and lifecycle policy (#6–#10) together before committing to schema changes; otherwise order amendments and historical presentation will need a second redesign. #15/#16 can retain documented current policy initially.

For destructive or broad mutations, add a domain-owned impact query reporting required/optional/explicit/implicit dependencies, published menus, open orders, stock/reservations, and proposed actions. This is especially useful for retire/replace and withdrawal. The command must recompute or validate that plan inside its transaction; a preview is never authorization or a concurrency guarantee. Keep initiating-action authorization explicit: consistency handlers should act on all required dependents, not silently omit entities hidden by user-facing list filters. Restrict disclosure in preview/report surfaces separately.

Migration must preserve known history without fabricating it. Existing aggregate ingredient snapshots cannot recover per-line substitutions, earlier recipe steps, old menu prices, or earlier names. Mark backfilled descriptions as legacy/current-derived, retain unknown values where necessary, and migrate mixed-unit reservations using their own stored units. Test restart and old-record reads, not just freshly seeded fixtures. Durable replacement relationships and reasons must be stored explicitly if they are needed after the transient event is gone.

Validation performed for this review:

- `go test ./app/... ./pkg/middleware ./pkg/dispatcher ./pkg/store ./architecture` passed (native linker emitted duplicate-library warnings).
- Six temporary diagnostic tests passed while asserting the observed defects: cancellation recovery, unit conversion, drink deletion through Update, implicit-substitute invalidation, handler-order dependence, and hidden order reservations. These were characterization checks, not claims that the behavior is correct; temporary test source was removed after the review.
- All remaining findings are based on the cited implementation paths and existing tests. No production implementation or migration was changed.
