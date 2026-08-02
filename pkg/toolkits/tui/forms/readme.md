# TUI forms

`forms` provides Bubble Tea form orchestration plus text, number, and select fields.

- `Form` owns field focus, explicit edit/accept mode, dirty state, validation, sizing, and key
  routing. The enclosing workflow still owns submit/cancel effects.
- `Field` is the extension contract. Built-ins are `TextField`, `NumberField`, and `SelectField`.
- `FieldOption` values configure initial values, required/length/range rules, precision,
  placeholders, negative numbers, and custom validators.
- `Validator` helpers cover required, length, numeric range, and regular-expression rules.
- `FormKeys`, `FormStyles`, and `FieldStyles` keep application policy injectable.

Construct fields, pass application styles/keys to `New`, forward messages while the form owns
interaction, call `Validate` before a command, and read typed values from the fields. See
`app/domains/ingredients/surfaces/tui/create_vm.go` for the basic lifecycle and the drinks recipe
editor for custom field composition.
