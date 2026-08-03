# CLI toolkit

This package is the small, application-independent edge of the urfave/cli surface. It standardizes
JSON mutation input and structured output; the nested [table package](table/readme.md) handles
human-readable output.

## Public surface

- `JSONFlag`, `TemplateFlag`, `StdinFlag`, and `FileFlag` give mutation commands consistent flags.
- `ReadJSONInput[T]` requires exactly one source: `--file` or `--stdin` (which may receive a pipe).
  The composing command handles `--template` before calling the reader.
- `WriteJSON` writes indented JSON with a trailing newline.

The domain CLI packages define transport-shaped view structs and validators. `main/cli` composes
those commands and selects `WriteJSON`, `table.PrintTable`, or `table.PrintDetail` according to the
command and `--json`; domain commands and queries never depend on this package.

## Fast path

Trace `main/cli/ingredients.go` for input/output selection, then
`app/domains/ingredients/surfaces/cli` for domain adaptation. For a new command:

1. Put business behavior on the domain module.
2. Add domain-specific flags, validation, and view conversion in its `surfaces/cli` package.
3. Compose the command in `main/cli`; use these shared flags/readers and the table package.
4. Cover text and JSON output in entrypoint tests.

Run `go test ./pkg/toolkits/cli/... ./main/cli` while iterating.
