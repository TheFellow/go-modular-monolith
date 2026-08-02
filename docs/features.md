# Application features

All CLI examples assume `go run ./main/seed` has created `data/mixology.db`. Substitute
`go run ./main/cli` for `mixology` when running from source.

## Filtering and paging

Every list accepts `--filter`; `--filter-help` prints its concrete fields, types, syntax, and
examples without opening the database. Expressions support comparisons, `in`/`not in`, parentheses,
`&&`/`and`, `||`/`or`, `!`/`not`, and string predicates such as `contains`, `startsWith`,
`endsWith`, and `matches`.

```sh
mixology drinks list --filter-help
mixology drinks list --filter '(category == "cocktail" || category == "tiki") && name.contains("rum")'
mixology inventory list --filter 'quantity <= 5 && unit in ["ml", "oz"]'
mixology audit list --limit 20 --cursor aud-...
```

Schemas live beside public domain models. Parsing yields an application-owned tree: supported
conjunctions are pushed into bstore while the full residual expression preserves exact semantics.
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

## Runtime configuration

CLI, TUI, GUI, and seeder default to `data/mixology.db`; only one process can own the embedded file.
Interactive entrypoints share `--db`, `--actor`, `--log-level`, `--log-format`, `--log-file`, and
`--metrics`, with corresponding `MIXOLOGY_*` variables. The GUI adds `--data-dir`. Command-line
options override environment values.
