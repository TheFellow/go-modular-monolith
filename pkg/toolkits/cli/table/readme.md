# CLI table rendering

`table` renders structs for terminal users while keeping rendering independent of Mixology models.

- `PrintTable` renders a slice as aligned columns.
- `PrintDetail` renders one struct as key/value rows.
- Tables opt exported fields in with `table:"Heading"`. Details use exported `json` field names and
  honor `omitempty`. Values use consistent time, `fmt.Stringer`, pointer, and scalar formatting, so
  adapters should flatten complex domain values into display-ready view structs first.

The usual flow is domain model → `app/domains/*/surfaces/cli` view → this renderer. See
`main/cli/ingredients.go` and its output tests for the shortest complete example.
