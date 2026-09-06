# Transactional domain workflows

This document records the implemented decisions from the [cross-domain review](cross-domain-mutation-review.md). These changes target a freshly seeded teaching database; there is no data migration or invented historical backfill.

## Domain ownership and preparation

Every command, its leaf handlers, and its successful audit activity share one SQL transaction. For each event, every `Handling` runs before any `Handle`. A handler that depends on another context's state prepares its result during `Handling`; `Handle` persists the prepared result in its own context. Menu handlers project reservation changes and recipe retirement before saving availability. Recipe retirement uses the same public pure domain rule in Drinks and in Menu preparation. No extra dispatch phases or cascading events are introduced.

Menu projections currently scan active draft and published menus. This deliberately includes implicit substitution dependencies rather than maintaining an incomplete reverse index. Each drink is calculated once per event preparation; publication refresh is restricted to its menu. Only changed menus are written; inspected menus are audit participants.

## Decisions and implementation

| Review | Implemented contract |
| --- | --- |
| 1 | Active Drink, Ingredient, Menu and Order models have no deletion timestamp. Catalog deletion metadata stays private; placement allowlists request fields and owns lifecycle initialization. |
| 2 | Stock and reservations persist canonical quantities (`ml` for volume, native discrete units otherwise). Display and cost units are separate. Compatible catalog unit changes update stock presentation, leaving price basis and accepted usage unchanged. Dimensional changes require a replacement identity. |
| 3 | Cancellation and amendment release reconcile other blocked orders. Completion and amendment verify reservation identity and quantities; completion refuses inconsistent or quarantined stock. |
| 4 | Stateful `Handling` preparation makes reciprocal reactions independent of sibling execution order. |
| 5 | Temporary substitution rules persist original/substitute IDs, ratio, quality, enabled state and revision. Renames have no identity effect. Stock changes and rule changes refresh dependent availability. |
| 6 | Acceptance captures menu/name, ordered lines and notes, agreed prices, recipe instructions, garnish, and chosen/omitted ingredients. Historical rendering uses that snapshot. |
| 7 | Explicit amendments preserve acceptance and agreed prices. The current plan and append-only amendment records retain actor, reason, quantities, ratios, and optional approved preparation changes. Terminal orders cannot be amended. Selected batches are atomic. |
| 8 | Discontinuation honors existing reservations while excluding stock from new service. Withdrawal quarantines stock and blocks affected orders. Release restores eligibility according to whether the ingredient remains active. Disposal records physical loss; quantity, identity, tags, and movement history remain. |
| 9 | Drink removal is blocked by any active menu or historical order usage. Menu removal is blocked by order usage. Errors identify dependencies and corrective action; no force option. Redrafting retains the previous publication timestamp. |
| 10 | Audit distinguishes changed entities, referenced participants, and domain-authored before/after effects. Composed commands share a workflow ID. A failed outer workflow rolls back child successes and persists its attempted effects as one failed activity. Cancellation includes terminal time and optional reason. |
| 11 | Recipe and substitute references must exist and be dimensionally compatible. Units/categories and finite positive recipe amounts/ratios are validated. Retirement that empties a recipe leaves it requiring review. |
| 12 | Authoritative readiness, fulfillment and persisted menu projections propagate dependency errors. Presentation-only fallback remains explicitly separate. |
| 13 | Costing and fulfillment use the same whole-recipe stock allocation planner. Omitted optionals cost nothing; margin calculations require matching currencies. |
| 14 | Every SQL update/delete requires a revision and compares it in SQL. Full stock sets require the expected revision. GUI/TUI edits carry captured revisions and expected complete tag sets, including composed edits. ID-only transitions load and CAS current state; supplied transition revisions must match. |
| 15 | A shortage blocks every affected open order. Replenishment, cancellation, amendment release and quarantine release reconcile that policy; no priority allocation. |
| 16 | Optional ingredients are included when stock permits, snapshotted, reserved and consumed. The planner can omit an optional to fulfill a required ingredient. Omission is retained in acceptance; it never silently consumes stock. |

## Explicit workflows

`Orders.Amend` accepts an order ID, expected revision, reason, and explicit replacements. A replacement's `OriginalID` identifies an ingredient in the current approved plan, including a substitute already selected at acceptance or by an earlier amendment. Ratios multiply the current selected quantity. Optional `Preparation` changes replace instructions/garnish for selected drink lines. Unspecified instructions remain as previously approved. An ID-only recipe substitute candidate cannot express a new quantity ratio: a non-1 permanent replacement affecting such a candidate is blocked with the dependent drink ID until that candidate is explicitly revised. This avoids silently using the wrong replacement quantity.

`App.AmendOrders` validates the selection's revisions before applying any changes, then commits every selected amendment or none. `App.RetireIngredient` composes that explicit selection with catalog retirement and its recipe/menu/stock reactions. Unselected orders retain their accepted plans and follow the chosen discontinuation or withdrawal policy. Amendment never implies customer consent; the operator supplies the approved selection and reason.

Independent CLI examples (replace identifiers and revisions with values read from your database):

```sh
mixology ingredients substitution --id ing-original --substitute-id ing-replacement --ratio 0.75 --quality similar
mixology ingredients substitutions --id ing-original
mixology ingredients retire --id ing-original --replacement-id ing-replacement --reason 'discontinued'
mixology orders amend --id ord-example --ingredient-id ing-original --replacement-id ing-replacement --ratio 0.75 --revision 3 --reason 'approved replacement'
mixology orders amend-batch --file amendments.json
mixology inventory quarantine --ingredient-id ing-original --reason 'inspection required'
mixology inventory release --ingredient-id ing-original --reason 'inspection passed'
mixology inventory dispose --ingredient-id ing-original --quantity 5 --reason 'discard remainder'
mixology inventory history --ingredient-id ing-original
```

Batch input is a JSON array of `orders/models.Amendment`, including each expected revision. This command accepts the domain model's JSON shape: typed IDs are objects with `Type` and `ID`, unlike the string IDs used by CLI flags. Replace the example IDs and revision with values from your database:

```json
[
  {
    "OrderID": {"Type": "Mixology::Order", "ID": "ord-example"},
    "Revision": 3,
    "Reason": "approved replacement and preparation",
    "Replacements": [
      {
        "OriginalID": {"Type": "Mixology::Ingredient", "ID": "ing-original"},
        "ReplacementID": {"Type": "Mixology::Ingredient", "ID": "ing-replacement"},
        "Ratio": 0.75
      }
    ],
    "Preparation": [
      {
        "DrinkID": {"Type": "Mixology::Drink", "ID": "drk-example"},
        "Steps": ["Use the approved replacement", "Stir and serve"]
      }
    ]
  }
]
```

The CLI exposes single and batch amendments plus stock disposition/history commands. GUI/TUI details display accepted history, approved preparation, and amendment reasons, but do not yet have amendment or stock disposition forms. Atomic retirement together with selected amendments is exposed through `App.RetireIngredient`; separate CLI commands cannot share its transaction.

Substitution-rule creation uses revision zero. Revising or disabling an existing rule requires its current revision from `ingredients substitutions --id ...`; omitted revision does not mean unconditional replacement. Amendment batches likewise require each current order revision and reject duplicate order IDs.

Cost values are always interpreted with `CostUnit`, independently of display quantity. A new explicit price defaults to the catalog unit unless a cost unit is supplied. Retaining a price also retains its basis. Stock and lifecycle commands accept explicit revision tokens; CLI operations that omit them load the current token immediately before writing.

Application compositions should use `middleware.RunWorkflow` as their outer boundary. If passed a caller-owned transaction, it participates in that transaction; the caller owns rollback and post-rollback failure recording. Successful child audit records never commit independently of business writes.

## Verification

Integration regressions cover retirement and cancellation handler permutations, reservation recovery, canonical unit changes, renamed substitution identities, immutable acceptance, approved preparation changes, optional stock accounting, stale stock/tag editors, selected-batch failure, late rollback, quarantine/release, retained disposal history, and deletion vetoes. Architecture checks require revision fields on every registered domain record. Store tests exercise stale update and delete rejection across independent database connections.
