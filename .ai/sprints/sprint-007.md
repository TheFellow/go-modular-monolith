# Sprint 007: Uniform Error Handling

## Goal

Create a consistent error handling package with domain-specific error types, generated from a list of error kind structs that include metadata for multi-surface mapping.

## Tasks

- [x] Create `pkg/errors/errors.go` with ErrorKind struct and error kind definitions
- [x] Create `pkg/errors/generate.go` with `//go:generate` directive
- [x] Create `pkg/errors/gen.tmpl` template for generating error types
- [x] Run `go generate` to produce `pkg/errors/errors_gen.go`
- [x] Generated file includes: error structs, Invalidf/NotFoundf/Internalf constructors, IsInvalid/IsNotFound/IsInternal checkers
- [x] Write unit tests for error types and Is* functions
- [x] Update application to use generated errors where applicable; errors are wrapped for surface-friendly messaging

## Notes

Error kinds are structs with metadata for cross-surface mapping:
```go
// pkg/errors/errors.go

type ErrorKind struct {
    Name     string // "Invalid", "NotFound", "Internal"
    Message  string // default message
    HTTPCode int    // 400, 404, 500
    GRPCCode int    // codes.InvalidArgument, codes.NotFound, codes.Internal
}

var (
    ErrInvalid = ErrorKind{
        Name:     "Invalid",
        Message:  "invalid",
        HTTPCode: 400,
        GRPCCode: 3,  // codes.InvalidArgument
    }
    ErrNotFound = ErrorKind{
        Name:     "NotFound",
        Message:  "not found",
        HTTPCode: 404,
        GRPCCode: 5,  // codes.NotFound
    }
    ErrInternal = ErrorKind{
        Name:     "Internal",
        Message:  "internal error",
        HTTPCode: 500,
        GRPCCode: 13, // codes.Internal
    }

    // Slice for codegen iteration
    ErrorKinds = []ErrorKind{ErrInvalid, ErrNotFound, ErrInternal}
)
```

Code generation produces constructors and checkers:
```go
// pkg/errors/errors_gen.go (generated)

// Creating errors - f suffix for formatting
errors.Invalidf("name is required")
errors.Invalidf("name is required: %w", err)
errors.NotFoundf("drink %s not found", id)
errors.Internalf("unexpected database error: %w", err)

// Checking errors (handles wrapped errors via errors.As)
if errors.IsInvalid(err) { ... }
if errors.IsNotFound(err) { ... }
if errors.IsInternal(err) { ... }

// Future: surface-specific helpers
err.HTTPCode() // returns 400, 404, 500
err.GRPCCode() // returns codes.InvalidArgument, etc.
```

Each generated error type:
- References its ErrorKind (for code lookups)
- Implements `Unwrap()` for error chain support
- Has an `f`-suffixed constructor for formatted messages
- Can expose HTTP/gRPC codes for transport layers

## Success Criteria

- `go generate ./pkg/errors/...` produces `errors_gen.go`
- `go test ./pkg/errors/...` passes
- Error kinds expose correct HTTP/gRPC codes
- Wrapped errors are correctly identified by Is* functions
- Wrapped errors can be cleanly converted to their pkg/errors type using errors.As() at which point they can be translated to any supported surface

## Dependencies

- Sprint 006 (working application)
