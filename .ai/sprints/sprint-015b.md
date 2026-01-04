# Sprint 015b: Optional Package (Intermezzo)

## Goal

Create a `pkg/optional` package using generics to represent optional values with exhaustive type switch support.

## Tasks

- [x] Create `pkg/optional/optional.go` with `Value[T]` interface
- [x] Implement `Some[T]` and `None[T]` concrete types
- [x] Add constructor functions `Some(v)` and `None[T]()`
- [x] Add methods: `IsSome()`, `IsNone()`, `Unwrap()`, `Must()`, `Or()`, `OrElse()`
- [x] Add functions: `Map()`, `FlatMap()` (type-changing operations)
- [x] Add `go-check-sumtype` to tool dependencies and CI
- [x] Add tests for all functionality
- [x] Update existing models using `*T` pointers to use `optional.Value[T]`
- [x] Verify `go test ./...` passes

## Design

### Sealed Interface Pattern

Go doesn't have sum types, but we can approximate exhaustive matching using a sealed interface with an unexported method:

```go
// pkg/optional/optional.go
package optional

//sumtype:decl
// Value represents an optional value of type T.
// Use type switch for exhaustive matching:
//
//   switch v := opt.(type) {
//   case optional.Some[string]:
//       fmt.Println(v.Val)
//   case optional.None[string]:
//       fmt.Println("no value")
//   }
type Value[T any] interface {
    IsSome() bool
    IsNone() bool
    Unwrap() (T, bool)
    Must() T
    Or(fallback T) T
    OrElse(fallback func() T) T
    sealed() // unexported - only Some and None can implement
}

// Some represents a present value.
type Some[T any] struct {
    Val T
}

func (Some[T]) IsSome() bool             { return true }
func (Some[T]) IsNone() bool             { return false }
func (s Some[T]) Unwrap() (T, bool)      { return s.Val, true }
func (s Some[T]) Must() T                { return s.Val }
func (s Some[T]) Or(fallback T) T        { return s.Val }
func (s Some[T]) OrElse(f func() T) T    { return s.Val }
func (Some[T]) sealed()                  {}

// None represents an absent value.
type None[T any] struct{}

func (None[T]) IsSome() bool             { return false }
func (None[T]) IsNone() bool             { return true }
func (None[T]) Unwrap() (T, bool)        { var zero T; return zero, false }
func (None[T]) Must() T                  { panic("optional: called Must on None") }
func (n None[T]) Or(fallback T) T        { return fallback }
func (n None[T]) OrElse(f func() T) T    { return f() }
func (None[T]) sealed()                  {}
```

### Constructors

```go
// NewSome creates a Some containing the value.
func NewSome[T any](v T) Some[T] {
    return Some[T]{Value: v}
}

// NewNone creates a None of type T.
func NewNone[T any]() None[T] {
    return None[T]{}
}
```

### Type-Changing Functions

Map and FlatMap must be standalone functions because they change the type parameter:

```go
// Map transforms the value if present.
func Map[T, U any](v Value[T], f func(T) U) Value[U] {
    if s, ok := v.(Some[T]); ok {
        return NewSome(f(s.Val))
    }
    return NewNone[U]()
}

// FlatMap transforms the value if present, allowing the function to return None.
func FlatMap[T, U any](v Value[T], f func(T) Value[U]) Value[U] {
    if s, ok := v.(Some[T]); ok {
        return f(s.Val)
    }
    return NewNone[U]()
}
```

## Usage Examples

### Basic Usage

```go
var name optional.Value[string] = optional.NewSome("Margarita")

// Type switch (exhaustive)
switch v := name.(type) {
case optional.Some[string]:
    fmt.Printf("Drink name: %s\n", v.Val)
case optional.None[string]:
    fmt.Println("No name provided")
}

// Unwrap (method)
if n, ok := name.Unwrap(); ok {
    fmt.Println(n)
}

// Or with fallback (method)
displayName := name.Or("Unknown")

// OrElse with lazy fallback (method)
displayName := name.OrElse(func() string { return computeDefault() })

// Predicates (methods)
if name.IsSome() { ... }
if name.IsNone() { ... }
```

### In Models

```go
// app/menu/models/menu.go
type MenuItem struct {
    DrinkID     string
    DisplayName optional.Value[string]  // Override drink name
    Price       optional.Value[Price]   // Optional pricing
    Featured    bool
}

// Usage - method syntax reads naturally
func (i MenuItem) Name(drinkName string) string {
    return i.DisplayName.Or(drinkName)
}
```

### With Map/FlatMap

```go
// Transform optional value (function - changes type)
upperName := optional.Map(name, strings.ToUpper)

// Chain optional operations
func findDrink(id string) optional.Value[Drink] { ... }
func getRecipe(d Drink) optional.Value[Recipe] { ... }

recipe := optional.FlatMap(findDrink("abc"), getRecipe)
```

## Exhaustive Type Switch Enforcement

Use [go-check-sumtype](https://github.com/alecthomas/go-check-sumtype) to enforce exhaustive type switches at lint time.

### Tool Setup

Add to `go.mod`:
```go
tool github.com/alecthomas/go-check-sumtype
```

### Annotation

The `//sumtype:decl` comment marks the interface as a sum type:

```go
//sumtype:decl
type Value[T any] interface {
    // ...
    sealed()
}
```

### CI Integration

Add to build/lint order:
```bash
go generate ./...
go build ./...
go tool arch-lint
go tool go-check-sumtype ./...   # Exhaustive switch check
go test ./...
```

### What It Catches

```go
// ERROR: missing case optional.None[string]
switch v := opt.(type) {
case optional.Some[string]:
    fmt.Println(v.Val)
// Missing None case - go-check-sumtype reports error
}
```

## Why Not Pointers?

Pointers (`*T`) can represent optional values but have drawbacks:

| Concern | Pointer | optional.Value |
|---------|---------|----------------|
| Nil safety | Runtime panic | Compile-time type check |
| Intent | Ambiguous (optional? mutable?) | Explicit |
| Type switch | Not possible | Exhaustive matching |
| Zero value | nil (valid None) | Must construct |

`optional.Value[T]` makes optionality explicit in the type system.

## JSON Serialization

Optional values serialize naturally:

```go
func (s Some[T]) MarshalJSON() ([]byte, error) {
    return json.Marshal(s.Value)
}

func (n None[T]) MarshalJSON() ([]byte, error) {
    return []byte("null"), nil
}
```

## Models to Update

Replace pointer-based optionality with `optional.Value[T]`:

| Model | Field | Before | After |
|-------|-------|--------|-------|
| `menu.Menu` | `PublishedAt` | `*time.Time` | `optional.Value[time.Time]` |
| `menu.MenuItem` | `DisplayName` | `*string` | `optional.Value[string]` |
| `menu.MenuItem` | `Price` | `*Price` | `optional.Value[Price]` |

**Before:**
```go
type Menu struct {
    PublishedAt *time.Time
}

// Usage - nil checks everywhere
if m.PublishedAt != nil {
    fmt.Println(*m.PublishedAt)
}
```

**After:**
```go
type Menu struct {
    PublishedAt optional.Value[time.Time]
}

// Usage - method syntax, no dereferencing
if t, ok := m.PublishedAt.Unwrap(); ok {
    fmt.Println(t)
}
// Or with type switch for exhaustive handling
```

## Success Criteria

- `optional.Value[T]` interface with sealed pattern and `//sumtype:decl` annotation
- `Some[T]` and `None[T]` concrete types with methods
- Methods: `IsSome()`, `IsNone()`, `Unwrap()`, `Must()`, `Or()`, `OrElse()`
- Functions: `Map()`, `FlatMap()` (type-changing)
- `go-check-sumtype` added to tools and CI pipeline
- `go tool go-check-sumtype ./...` passes (all type switches exhaustive)
- All models using `*T` for optionality updated to `optional.Value[T]`
- Comprehensive tests
- `go test ./...` passes

## Dependencies

- Sprint 015 (uses optional in models)
