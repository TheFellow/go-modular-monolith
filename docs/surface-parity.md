# Cross-domain surface parity

The CLI, terminal application, and desktop application expose the same transactional domain
operations. Presentation adapters retain the revision shown to the operator and call the public
application modules; they do not duplicate reservation, retirement, or fulfillment rules.

## Audit scope

This audit follows the contracts in [Transactional workflows](transactional-workflows.md),
including the September 2026 changes to substitutions, accepted order history, stock retention,
catalog deletion, audit effects, and optimistic concurrency.

| Contract | CLI | TUI and GUI |
| --- | --- | --- |
| Substitution rules | `ingredients substitution` / `substitutions` | List enabled and disabled rules; create, revise, disable, and re-enable with captured rule revisions. |
| Ingredient retirement | `ingredients retire` | Choose permanent replacement and ratio, withdrawal, and reason. |
| Approved fulfillment | `orders amend` / `amend-batch` | Amend an open order, queue selected amendments, review and commit the batch atomically. |
| Order history | `orders get` / `--json` | Accepted names, prices, recipes and omissions; current approved plan; amendment before/after records; cancellation metadata. |
| Retained stock | `inventory quarantine` / `release` / `dispose` / `history` | Matching lifecycle forms and movement history, including stock whose ingredient was retired. |
| Quantity and cost basis | Stock command quantities and `--cost-unit` | Quantity labels use the stock display unit; the price has its own explicit cost unit. |
| Initial stock receipt | `inventory set` for a new ingredient | Receive new stock from the inventory list; a concurrent first receipt cannot be overwritten. |
| Catalog constraints | Domain dependency errors | The same command constraints, with retirement-aware recipe details and accurate deletion feedback. |
| Menu readiness and analysis | `menus readiness` / `show --costs` | Publication blockers, degraded published menus, and substitution details in analysis. |
| Transaction evidence | Audit `--details` / `--json` | Command activity, referenced entities, and labeled field changes; failed effects are identified as attempts. |

Order amendment batches call `Orders.AmendBatch` and retain every selected order's expected
revision. Catalog retirement calls `Ingredients.Retire` separately. Tagged forms pass `tag.Edit`
to the owning command; Tagging's leaf reaction shares that command's transaction and activity.

## Review captures

The opt-in render tests exercise real widgets and the complete terminal shell against isolated
application fixtures. They write review artifacts without requiring platform-specific pixel goldens:

```sh
MIXOLOGY_RENDER_DIR=/tmp/mixology-review go test -tags ci ./main/gui \
  -run '^TestRenderWorkspaceReview$' -count=1
MIXOLOGY_RENDER_DIR=/tmp/mixology-review go test ./main/tui \
  -run '^TestRenderCrossDomainReview$' -count=1
```

Desktop captures are PNG files; terminal captures preserve the rendered frame as `.ansi` text.
Review the workflow forms together with the surrounding list/detail pages, then inspect the
retirement, stock history, and accepted-order views after mutations. Behavioral tests exercise
permissions, captured revisions, batch rollback, keyboard input, and semantic widget controls.
Run the [full CI check](development.md#full-ci-check) before publishing changes.
