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

All list commands share paging and typed filter expressions. Mutation commands that accept a JSON
document use `--file` or `--stdin` (which may receive a pipe); `--template` prints their expected shape.
See the [feature guide](../../docs/features.md) for IDs, filters, tags, authorization personas, and
audit examples.

Replace-style update documents for drinks, ingredients, and menus must round-trip the positive
`revision` returned by a read. The value is an opaque concurrency token: do not increment it in a
script. A stale value returns the standard conflict exit code instead of overwriting another
client's change. Flag-based ingredient and menu updates read the current revision immediately
before submitting, while JSON input remains explicit so read/edit/write automation can detect
concurrent changes.

Retirement is a distinct authorized operation; `ingredients delete` remains a compatibility alias
for retirement without replacement. An explicit replacement updates compatible future recipes,
while omission preserves affected recipes for review. `menus readiness` reports publication
blockers and warnings to authorized operators. Existing published menus may degrade in place, but
the publish command rejects a draft with known blockers. These commands expose the same domain
rules, event reactions, and audit touches as the TUI and GUI.

## Adding a command

1. Expose the operation through the owning domain's public module.
2. Add transport validation and a display view in `app/domains/<domain>/surfaces/cli`.
3. Compose flags and output in the domain-named file here, using `pkg/toolkits/cli`.
4. Add entrypoint tests for text/JSON output, failures, and any global-option interaction.

Run `go test ./pkg/toolkits/cli/... ./app/domains/<domain>/surfaces/cli ./main/cli`.
