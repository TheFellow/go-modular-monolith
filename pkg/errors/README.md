# Application errors

`pkg/errors` classifies application failures independently of presentation or transport. Domain,
persistence, middleware, and presentation code can return one typed error while CLI, TUI, GUI,
HTTP, or gRPC edges derive consistent behavior from its immutable kind. Diagnostic detail remains
available for logs and wrapping, while presentation adapters use safe user-facing text.

## Error kinds

| Kind                 | Default message       | HTTP | gRPC                 | CLI | TUI style |
| -------------------- | --------------------- | ---: | -------------------- | --: | --------- |
| `Invalid`            | `invalid`             |  400 | `InvalidArgument`    |  10 | error     |
| `NotFound`           | `not found`           |  404 | `NotFound`           |  20 | warning   |
| `Permission`         | `permission denied`   |  403 | `PermissionDenied`   |  30 | error     |
| `Conflict`           | `conflict`            |  409 | `AlreadyExists`      |  40 | warning   |
| `FailedPrecondition` | `failed precondition` |  412 | `FailedPrecondition` |  45 | warning   |
| `Internal`           | `internal error`      |  500 | `Internal`           |  50 | error     |

`Kind`, `SpecFor`, and `AllKinds` expose this mapping. `SpecFor` returns a copy, and an unknown kind
falls back to the internal-error specification. HTTP and gRPC values are metadata; this package
does not start a server or write a response.

## Constructing and inspecting errors

Use the generated constructor that describes the failure, and include actionable context:

```go
if name == "" {
	return errors.Invalidf("name is required")
}

if _, ok := catalog[id]; !ok {
	return errors.NotFoundf("drink %s not found", id)
}
```

Each constructor returns a distinct typed error such as `*InvalidError` or `*NotFoundError`.
Classification traverses ordinary wrapping:

```go
if errors.IsNotFound(err) {
	// recover, translate, or assert the expected failure
}

var appErr *errors.Error
if errors.As(err, &appErr) {
	log.Printf("kind=%s detail=%s", appErr.Kind(), appErr.Error())
}
```

The package forwards `As`, `AsType`, `Is`, `Join`, `New`, and `Unwrap` from the standard library so
the rest of the repository imports only `pkg/errors`, using its natural `errors` package name.
Constructors accept `fmt.Errorf`-style formatting and retain a single `%w` cause for
`errors.Is`/`errors.As` traversal. The architecture lint configuration rejects direct standard
library `errors` imports outside this package.

Choose kinds by meaning rather than by the current surface:

- `Invalid` means the request or input is malformed.
- `NotFound` means the requested application resource does not exist.
- `Permission` means the actor is not allowed to perform the operation.
- `Conflict` means the request collides with existing state, such as a duplicate.
- `FailedPrecondition` means current state does not permit an otherwise valid operation.
- `Internal` means an invariant, dependency, evaluation, or unexpected implementation failure.

For example, the [authorization evaluator](../authz/README.md#evaluation-api) returns `Permission`
for a Cedar denial and `Internal` when a resource or evaluation is invalid.

## Diagnostic detail and safe presentation

`Error()` returns diagnostic detail. For every non-internal kind, `UserMessage()` returns that same
actionable detail by default. Internal errors instead return the generic `internal error` message,
preventing implementation details such as credentials or storage failures from reaching a user.
`WithUserMessage` supplies an explicit safe override when a better recovery message is available:

```go
return errors.Internalf("load inventory: %w", err).
	WithUserMessage("Inventory is temporarily unavailable; please try again")
```

Do not render `Error()` directly at a presentation boundary when an application error may contain
sensitive detail. Also classify unexpected failures as `Internal`; unknown errors do not have the
package's safe-message guarantee.

## Presentation adapters

- `ToCLIExit` converts application errors to `urfave/cli` exit errors with the mapped code and safe
  message. It preserves an existing `cli.ExitCoder`; an otherwise unknown error uses exit code 1.
- `ToTUIError` returns the mapped style, safe message, and original cause. Unknown errors use error
  styling and their original message; `nil` produces an informational empty value.
- The [GUI toolkit](../toolkits/gui/readme.md) maps the same kinds to inline, warning, and error
  presentation through `PresentError`.

The [CLI entrypoint](../../main/cli/README.md) applies `ToCLIExit` at process and command boundaries;
the [TUI entrypoint](../../main/tui/README.md) uses `ToTUIError` for its status bar. Lower layers
should return typed errors without choosing colors, dialogs, process behavior, or transport output.

## Adding an error kind

Kinds are a generated family. To add one:

1. Add its `Kind` constant, `Spec`, and `allKinds` entry in [`kind.go`](kind.go).
2. Run `go generate ./...` from the repository root.
3. Commit the regenerated [`errors_gen.go`](errors_gen.go) and
   [`pkg/testutil/errors_gen.go`](../testutil/errors_gen.go).
4. Extend mapping, safe-message, wrapping, and surface-adapter tests.

Generation creates the typed constructor, classifier, metadata methods, and matching test helper;
generated files should not be edited directly. Run `go test ./pkg/errors ./pkg/testutil` while
iterating. Repository-wide error and generation boundaries are summarized in the
[architecture guide](../../docs/architecture.md#generation).
