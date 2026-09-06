# CLI entrypoint

`main/cli` composes the `mixology` executable. It owns process/runtime configuration, the root
urfave/cli command tree, selection between JSON and human output, and conversion of returned errors
to process exit behavior through the shared [application error mapping](../../pkg/errors/README.md).
Business rules remain in domain modules; reusable input and rendering mechanics remain in the
[CLI toolkit](../../pkg/toolkits/cli/readme.md).

## Request path

```text
main.go -> command composition -> app.New + fresh middleware context
        -> app/domains/<domain>/surfaces/cli
        -> domain module command/query -> middleware -> persistence
        -> CLI view -> JSON or table renderer
```

Each invocation creates fresh operation context, so actor, logging, metrics, authorization, unit of
work, event dispatch, and audit behavior match the interactive surfaces. `cli.go` owns global
options; domain-named files compose their command groups. Domain surface packages own view structs,
parsing rules, filter help, and domain-specific validation.

## Run and discover

```sh
go run ./main/seed
go run ./main/cli --help
go run ./main/cli ingredients list --filter-help
go run ./main/cli ingredients list --limit 20 --json
go run ./main/cli --actor bartender menus list
go run ./main/cli ingredients retire --id ing-old --replacement-id ing-new --replacement-ratio 1
go run ./main/cli --actor manager menus readiness --id mnu-example
```

Operational domain lists share paging and typed filter expressions; substitution-rule and movement
history commands return their own collections. CRUD document commands use `--file` or `--stdin`
(which may receive a pipe); `--template` prints their expected shape. Amendment batches use the same
input reader with a JSON array and have no template flag.
See the [feature guide](../../docs/features.md) for IDs, filters, tags, authorization personas, and
audit examples.

Replace-style update documents for drinks, ingredients, and menus must round-trip the positive
`revision` returned by a read. The value is an opaque concurrency token: do not increment it in a
script. A stale value returns the standard conflict exit code instead of overwriting another
client's change. Flag-based ingredient and menu updates read the current revision immediately
before submitting, while JSON input remains explicit so read/edit/write automation can detect
concurrent changes.

Retirement is a distinct authorized operation; `ingredients delete` remains a compatibility alias
for the same retirement command. An explicit replacement updates compatible future recipes,
while omission leaves required references under review and removes optional/substitute references.
Existing stock and accepted reservations are retained; `--withdraw` quarantines stock and blocks
affected open orders. `menus readiness` reports publication
blockers and warnings to authorized operators. Existing published menus may degrade in place, but
the publish command rejects a draft with known blockers. These commands expose the same domain
rules and transactional audit effects as the TUI and GUI.

## Amendments, substitutions, and stock history

| Command | Contract |
| --- | --- |
| `ingredients substitution` | Create a rule with revision zero, or supply the rule's current `--revision` to revise/disable it. `ingredients substitutions --id ...` lists enabled and disabled rules. |
| `orders amend` | Replace a currently selected ingredient using `--ingredient-id`, `--replacement-id`, `--ratio`, and a required `--reason`. Acceptance and agreed prices remain unchanged. |
| `orders amend-batch` | Read an array of amendments from `--file` or `--stdin`; every entry needs its expected revision. The complete selection commits or rolls back together. |
| `inventory quarantine` / `release` | Change retained stock eligibility with `--ingredient-id` and a required `--reason`. Release respects catalog retirement. |
| `inventory dispose` | Record physical disposal of discontinued/quarantined stock using `--quantity` in its display unit and a required `--reason`. |
| `inventory history` | Read retained physical stock movements by `--ingredient-id`. |

These commands return JSON directly. Stock set/disposition and single-order amendment commands can
load the current revision when it is omitted; read/edit/write automation should supply the token
it originally read. Rule revisions and amendment-batch revisions are explicit. An order amendment's
ingredient ID identifies the ingredient in the current plan, which may already differ from the
original acceptance after an earlier substitution or amendment.

The CLI exposes retirement and amendment batches as separate commands. To retire an ingredient and
amend selected orders in one transaction, call `App.RetireIngredient`; running two CLI invocations
does not create an atomic workflow. See the [workflow examples and batch shape](../../docs/transactional-workflows.md#explicit-workflows).

## Adding a command

1. Expose the operation through the owning domain's public module.
2. Add transport validation and a display view in `app/domains/<domain>/surfaces/cli`.
3. Compose flags and output in the domain-named file here, using `pkg/toolkits/cli`.
4. Add entrypoint tests for text/JSON output, failures, and any global-option interaction.

Run `go test ./pkg/toolkits/cli/... ./app/domains/<domain>/surfaces/cli ./main/cli`.
