# Sprint 008: Decentralized Policy Embedding

## Goal

Refactor Cedar policy embedding so each module owns its embedded policies, and `pkg/authz` aggregates them by reference rather than copying.

## Tasks

- [x] Create `app/drinks/authz/policies.go` that embeds `policies.cedar` via `//go:embed`
- [x] Export embedded policy as public var (`var Policies string`)
- [x] Update `pkg/authz/policies.go` to embed `base.cedar` and define `PolicyDocument`
- [x] Remove codegen that copies policy files into a map
- [x] Update policy codegen to generate imports + document references
- [x] Ensure `go generate ./pkg/authz/...` produces updated `policies_gen.go`
- [x] Verify all policies load correctly at runtime (tests pass)

## Notes

Before (centralized copying):
```go
// pkg/authz/policies_gen.go (generated)
var policies = map[string]string{
    "drinks": "permit(...)",  // copied content
    "base":   "permit(...)",
}
```

After (decentralized embedding):
```go
// app/drinks/authz/policies.go
package authz

import _ "embed"

//go:embed policies.cedar
var Policies string
```

```go
// pkg/authz/policies.go
package authz

import _ "embed"

//go:embed base.cedar
var basePolicies string
```

```go
// pkg/authz/policies_gen.go (generated)
package authz

import (
    drinksauthz "github.com/TheFellow/go-modular-monolith/app/drinks/authz"
)

func policyDocuments() []PolicyDocument {
    return []PolicyDocument{
        {Name: "pkg/authz/base.cedar", Text: basePolicies},
        {Name: "app/drinks/authz/policies.cedar", Text: drinksauthz.Policies},
    }
}
```

Benefits:
- Each module owns its policies (better encapsulation)
- No file copying during codegen
- Policies are compile-time embedded where they're defined
- `pkg/authz` just aggregates references

## Success Criteria

- `go generate ./pkg/authz/...` produces updated aggregation code
- `go build ./...` passes
- `go test ./...` passes
- Cedar authorization still works correctly

## Dependencies

- Sprint 007 (uniform error handling)
