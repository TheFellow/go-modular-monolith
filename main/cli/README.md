# CLI entrypoint

`main/cli` composes the `mixology` executable. It owns process/runtime configuration, the root
urfave/cli command tree, selection between JSON and human output, and conversion of returned errors
to process exit behavior. Business rules remain in domain modules; reusable input and rendering
mechanics remain in the [CLI toolkit](../../pkg/toolkits/cli/readme.md).

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
```

All list commands share paging and typed filter expressions. Mutation commands that accept a JSON
document use `--file` or `--stdin` (which may receive a pipe); `--template` prints their expected shape.
See the [feature guide](../../docs/features.md) for IDs, filters, tags, authorization personas, and
audit examples.

## Adding a command

1. Expose the operation through the owning domain's public module.
2. Add transport validation and a display view in `app/domains/<domain>/surfaces/cli`.
3. Compose flags and output in the domain-named file here, using `pkg/toolkits/cli`.
4. Add entrypoint tests for text/JSON output, failures, and any global-option interaction.

Run `go test ./pkg/toolkits/cli/... ./app/domains/<domain>/surfaces/cli ./main/cli`.
