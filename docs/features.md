# Application features

All CLI examples assume `go run ./main/seed` has created `data/mixology.db`. Substitute
`go run ./main/cli` for `mixology` when running from source.

## Filtering and paging

Every list uses the shared [typed filter package](../pkg/filter/README.md) and accepts `--filter`;
`--filter-help` prints its concrete fields, types, syntax, and examples without opening the
database. Expressions support comparisons, `in`/`not in`, parentheses, `&&`/`and`, `||`/`or`,
`!`/`not`, and string predicates such as `contains`, `startsWith`, `endsWith`, and `matches`.

```sh
mixology drinks list --filter-help
mixology drinks list --filter '(category == "cocktail" || category == "tiki") && name.contains("rum")'
mixology inventory list --filter 'quantity <= 5 && unit in ["ml", "oz"]'
mixology audit list --limit 20 --cursor aud-...
```

Schemas live beside public domain models. Parsing yields an application-owned tree: supported
conjunctions are pushed into SQLite while the full residual expression preserves exact semantics.
Operational lists expose hydrated `tags`; match one canonical tag, not the serialized collection.

## Tags

Drinks, ingredients, inventory, menus, and orders accept label tags (`featured`) and key/value tags
(`region=west`). Audit entries are intentionally not taggable. Keys are case-sensitive, unique per
entity, and cannot contain `=` or controls; values may contain additional `=`. Collections use Go
CSV spelling and canonical key order.

```sh
mixology tags add drk-abc123 featured
mixology tags add drk-abc123 audience=sommelier
mixology tags list drk-abc123
mixology tags remove drk-abc123 audience
mixology tags show --key audience
mixology tags summary
mixology drinks list --filter 'tags contains "audience=sommelier"'
```

Adding an existing key replaces its value; repeated add and missing remove are successful no-ops.
Non-delete mutations also accept `--tags`: omission preserves, a value replaces the whole set, and
`--tags=` clears it. The domain mutation and tag replacement are one authorized, audited transaction.

Associations are centrally stored by Cedar entity type, ID, and key, but each owning domain loads
its own entities and hydrates tags in one type-scoped query. Soft-deleted rows retain associations;
the inventory hard-delete path removes them transactionally. Tags have no reserved application
meaning—individual Cedar policies or other consumers choose semantics.

Tag discovery returns each active match's domain-provided display name alongside its entity type
and ID. The CLI, TUI, and GUI render that same shared reference model; the tagging discovery
permission therefore governs disclosure of all three identity fields without requiring the owning
domain's separate read permission.

## Authorization personas

Use `--actor` (or `--as`) with any interactive entrypoint. Owner has full access; manager has broad
operational access; sommelier and bartender see role-appropriate catalogs/workflows; anonymous is
public read-only. Cedar can also filter individual list rows.

```sh
mixology --actor bartender menus list
mixology --as anonymous drinks list
```

## IDs and JSON

Typed IDs include a prefix: `drk-`, `ing-`, `inv-`, `mnu-`, `ord-`, and `aud-`. Primary IDs use
`--id`; references name the target (`--menu-id`, `--drink-id`, and so on). Cross-domain tag commands
infer the entity type from the five operational prefixes and reject audit IDs.

Add `--json` for structured output. Document mutations accept `--file` or `--stdin` (including a
pipe), and `--template` prints the expected JSON shape.

## Audit

Every write passing through `RunCommand` produces an activity. Handlers call `TouchEntity` for
indirectly affected entities. Successful activity records commit with the write; rejected attempts
are recorded after rollback.

```sh
mixology audit list --limit 20
mixology audit list --principal owner
mixology audit list --entity Mixology::Drink::drk-abc123
mixology audit list --filter 'principal.contains("owner") && success'
mixology audit history Mixology::Drink::drk-abc123
```

## Stateful fulfillment and retirement

Placing an order captures its ingredient-usage snapshot and reserves that stock in Inventory.
Inventory lists distinguish on-hand, reserved, and available quantities. Completing the order
consumes its reservations; cancelling releases them. A later stock correction below the reserved
total moves every affected pending order to `blocked`, and replenishment returns it to `pending`.
Blocked orders remain cancellable but cannot be completed.

This collaboration is deliberately reciprocal: Order events change Inventory reservations and
published Menu availability, while Inventory adjustment events change Order fulfillment state.
Every indirect mutation is part of the originating transaction and appears in its audit touches.

Retiring an ingredient (`ingredients retire`; `delete` remains a compatibility alias) removes its
stock but preserves dependent Drinks and Menu curation. A required canonical reference makes its
Drink `review_required`; an optional reference is removed, and a retired substitute candidate is
discarded. `--replacement-id` records explicit permanent product intent: category and unit
compatibility are validated and affected canonical recipes are rewritten transactionally. The
system never infers a permanent replacement from a temporary substitution rule.
Retirement has its own Cedar action and audit vocabulary; both the retired and replacement
ingredients are recorded as material participants in a successful replacement operation.
Category compatibility is intentionally coarse in this teaching model: the authorized manager's
explicit approval supplies the product-family judgment (for example, comparable tequila brands),
while the application prevents cross-category and dimensionally invalid rewrites.

Published Menus remain published when real-world changes degrade them. Their item availability and
the `menus readiness` report explain blockers and warnings. A temporary substitute can make a
review-required Drink limited rather than unavailable, but it remains a publication blocker until
the canonical recipe is approved. Draft publication rejects review-required Drinks, unavailable
items, retired/missing requirements, and temporary substitutions; ordinary low stock is a warning.
The manager-only report and blocked-action reason are available in CLI, TUI, and GUI without
exposing draft operational findings to read-only personas.

Editing a Drink with a valid replacement recipe returns it to `active`; existing Orders retain the
usage snapshot accepted when they were placed.
Pending Orders reserved against the retired ingredient become `blocked`; they preserve that
historical requirement and may still be cancelled to release the reservation.

## Runtime configuration

CLI, TUI, GUI, and seeder default to `data/mixology.db`. Multiple processes on the same machine may
open that local file: WAL permits concurrent readers, SQLite serializes writers, and a blocked
operation waits up to 10 seconds before returning a busy error. Do not place the database on a
network filesystem or share it between machines.
Interactive entrypoints share `--db`, `--actor`, `--log-level`, `--log-format`, `--log-file`, and
`--metrics`, with corresponding `MIXOLOGY_*` variables. The GUI adds `--data-dir`. Command-line
options override environment values. The [telemetry guide](../pkg/telemetry/README.md) documents
the metrics backends, Prometheus lifecycle, emitted instruments, and testing support.
