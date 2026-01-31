# Sprint 031d: KSUID Infrastructure

## Goal

Set up KSUID (K-Sortable Unique IDentifier) infrastructure for time-sortable entity IDs with optional type prefixes.

## Status

- Started: 2026-01-12
- Completed: 2026-01-13

## Why KSUIDs?

Current IDs are random hex strings with no inherent ordering:
```
a1b2c3d4e5f6g7h8  (random)
```

KSUIDs provide:
- **Time-sorted**: IDs generated later sort after earlier ones
- **Globally unique**: 20 bytes (4-byte timestamp + 16-byte random)
- **URL-safe**: Base62 encoded, 27 characters
- **No coordination**: No central authority needed

Example KSUID:
```
2HbR6c7bDtx9XPVqG1kN9MHHFM7
```

## Package Selection

**Recommended: `github.com/segmentio/ksuid`**

- Well-maintained by Segment
- Zero dependencies
- Battle-tested in production
- Simple API

```go
import "github.com/segmentio/ksuid"

id := ksuid.New()           // Generate new KSUID
str := id.String()          // "2HbR6c7bDtx9XPVqG1kN9MHHFM7"
ts := id.Time()             // Extract timestamp
parsed, _ := ksuid.Parse(str) // Parse from string
```

## ID Prefix Convention

For human readability and debugging, IDs should include a short type prefix:

| Entity Type | Prefix | Example ID |
|-------------|--------|------------|
| Drink | `drk-` | `drk-2HbR6c7bDtx9XPVqG1kN9MHHFM7` |
| Ingredient | `ing-` | `ing-2HbR6c7bDtx9XPVqG1kN9MHHFM7` |
| Menu | `mnu-` | `mnu-2HbR6c7bDtx9XPVqG1kN9MHHFM7` |
| Order | `ord-` | `ord-2HbR6c7bDtx9XPVqG1kN9MHHFM7` |
| Inventory | `inv-` | `inv-2HbR6c7bDtx9XPVqG1kN9MHHFM7` |
| AuditEntry | `aud-` | `aud-2HbR6c7bDtx9XPVqG1kN9MHHFM7` |

This makes IDs self-describing when seen in logs, URLs, or debugging.

## Updated `pkg/ids` Package

```go
// pkg/ids/ids.go
package ids

import (
    "fmt"
    "strings"

    "github.com/cedar-policy/cedar-go"
    "github.com/segmentio/ksuid"
)

// Prefix maps entity types to their ID prefixes
var prefixes = map[cedar.EntityType]string{
    "Mixology::Drink":      "drk",
    "Mixology::Ingredient": "ing",
    "Mixology::Menu":       "mnu",
    "Mixology::Order":      "ord",
    "Mixology::Inventory":  "inv",
    "Mixology::AuditEntry": "aud",
}

// New generates a new KSUID-based EntityUID with type prefix
func New(entityType cedar.EntityType) (cedar.EntityUID, error) {
    id := ksuid.New()

    prefix, ok := prefixes[entityType]
    if !ok {
        // Fallback: derive prefix from entity type
        prefix = derivePrefix(entityType)
    }

    idStr := fmt.Sprintf("%s-%s", prefix, id.String())
    return cedar.NewEntityUID(entityType, cedar.String(idStr)), nil
}

// Parse extracts the KSUID from a prefixed ID string
func Parse(idStr string) (ksuid.KSUID, error) {
    parts := strings.SplitN(idStr, "-", 2)
    if len(parts) != 2 {
        return ksuid.Nil, fmt.Errorf("invalid id format: %s", idStr)
    }
    return ksuid.Parse(parts[1])
}

// Time extracts the timestamp from a prefixed ID
func Time(idStr string) (time.Time, error) {
    id, err := Parse(idStr)
    if err != nil {
        return time.Time{}, err
    }
    return id.Time(), nil
}

func derivePrefix(entityType cedar.EntityType) string {
    s := string(entityType)
    if idx := strings.LastIndex(s, "::"); idx >= 0 {
        s = s[idx+2:]
    }
    s = strings.ToLower(s)
    if len(s) > 3 {
        s = s[:3]
    }
    return s
}
```

## Sorting Guarantee

KSUIDs are lexicographically sortable by time:

```go
id1 := ids.New(DrinkEntityType)  // drk-2HbR6c7bDtx9XPVqG1kN9MHHFM7
time.Sleep(time.Millisecond)
id2 := ids.New(DrinkEntityType)  // drk-2HbR6c7bDtx9XPVqG1kN9MHHFM8

// id1 < id2 lexicographically because of KSUID time component
// Database ORDER BY id ASC gives chronological order
```

This is especially valuable for audit entries where you want time-ordered retrieval by default.

## Tasks

- [x] Add `github.com/segmentio/ksuid` to `go.mod` (vendored)
- [x] Update `pkg/ids/ids.go` to use KSUID with prefixes
- [x] Keep the prefix list static (no `RegisterPrefix`)
- [x] Add `Parse()` and `Time()` helper functions
- [x] Add tests for ID generation and parsing
- [x] Verify `go test ./...` passes
- [x] Verify packages compile (via `go test`)

## Migration Notes

Existing entities with old-style IDs will continue to work - the Cedar EntityUID is just a string. New entities will get KSUID-based IDs. If full migration is needed, that's a separate data migration sprint.

## Acceptance Criteria

- [x] `ids.New()` generates KSUID-based IDs with type prefix
- [x] IDs are lexicographically sortable by creation time
- [x] `ids.Parse()` can extract KSUID from prefixed ID
- [x] `ids.Time()` can extract timestamp from ID
- [x] All existing tests pass
- [x] Prefix list is static; unknown types derive a deterministic prefix
