# Sprint 033i: Consistent CLI Flag Names for Primary Keys

## Goal

Standardize CLI flag naming so primary key IDs use `--id` while cross-entity references use entity-specific names.

## Problem

Inconsistent flag naming across CLI commands:

```bash
# Inconsistent - uses entity-prefixed name for PK
mixology menu show --menu-id mnu-abc123
mixology order get --order-id ord-abc123

# Consistent - uses --id for PK
mixology drink get --id drk-abc123
mixology ingredient get --id ing-abc123
```

The command context already establishes which entity we're working with, so `--menu-id` is redundant in `menu show`.

## Convention

| Scenario | Flag Name | Example |
|----------|-----------|---------|
| Primary key of command's entity | `--id` | `menu show --id mnu-abc` |
| Reference to another entity | `--<entity>-id` | `menu add-drink --id mnu-abc --drink-id drk-xyz` |
| Lookup key (not a PK) | `--<entity>-id` | `inventory get --ingredient-id ing-abc` |

## Current State

### Correct (no changes needed)

| File | Command | Flag | Reason |
|------|---------|------|--------|
| `ingredients.go` | get | `--id` | PK |
| `ingredients.go` | update | `--id` | PK |
| `ingredients.go` | delete | `--id` | PK |
| `drinks.go` | get | `--id` | PK |
| `drinks.go` | delete | `--id` | PK |
| `menu.go` | add-drink | `--menu-id`, `--drink-id` | Two entities |
| `menu.go` | remove-drink | `--menu-id`, `--drink-id` | Two entities |
| `orders.go` | place | `--menu-id` | Reference to menu, not order PK |
| `inventory.go` | get | `--ingredient-id` | Lookup by ingredient (no inventory PK) |
| `inventory.go` | adjust | `--ingredient-id` | Lookup by ingredient |
| `inventory.go` | set | `--ingredient-id` | Lookup by ingredient |

### Needs Update

| File | Command | Current | Should Be |
|------|---------|---------|-----------|
| `menu.go` | show | `--menu-id` | `--id` |
| `menu.go` | publish | `--menu-id` | `--id` |
| `orders.go` | get | `--order-id` | `--id` |
| `orders.go` | complete | `--order-id` | `--id` |
| `orders.go` | cancel | `--order-id` | `--id` |

## Changes

### menu.go

**show command (line 80):**
```go
// Before
&cli.StringFlag{Name: "menu-id", Usage: "Menu ID", Required: true},
// ...
menuID, err := entity.ParseMenuID(cmd.String("menu-id"))

// After
&cli.StringFlag{Name: "id", Usage: "Menu ID", Required: true},
// ...
menuID, err := entity.ParseMenuID(cmd.String("id"))
```

**publish command (line 250):**
```go
// Before
&cli.StringFlag{Name: "menu-id", Usage: "Menu ID", Required: true},
// ...
menuID, err := entity.ParseMenuID(cmd.String("menu-id"))

// After
&cli.StringFlag{Name: "id", Usage: "Menu ID", Required: true},
// ...
menuID, err := entity.ParseMenuID(cmd.String("id"))
```

### orders.go

**get command (line 102):**
```go
// Before
&cli.StringFlag{Name: "order-id", Usage: "Order ID", Required: true},
// ...
orderID, err := entity.ParseOrderID(cmd.String("order-id"))

// After
&cli.StringFlag{Name: "id", Usage: "Order ID", Required: true},
// ...
orderID, err := entity.ParseOrderID(cmd.String("id"))
```

**complete command (line 135):**
```go
// Before
&cli.StringFlag{Name: "order-id", Usage: "Order ID", Required: true},
// ...
orderID, err := entity.ParseOrderID(cmd.String("order-id"))

// After
&cli.StringFlag{Name: "id", Usage: "Order ID", Required: true},
// ...
orderID, err := entity.ParseOrderID(cmd.String("id"))
```

**cancel command (line 154):**
```go
// Before
&cli.StringFlag{Name: "order-id", Usage: "Order ID", Required: true},
// ...
orderID, err := entity.ParseOrderID(cmd.String("order-id"))

// After
&cli.StringFlag{Name: "id", Usage: "Order ID", Required: true},
// ...
orderID, err := entity.ParseOrderID(cmd.String("id"))
```

## Result

```bash
# After - consistent PK naming
mixology menu show --id mnu-abc123
mixology menu publish --id mnu-abc123
mixology order get --id ord-abc123
mixology order complete --id ord-abc123
mixology order cancel --id ord-abc123

# Multi-entity operations still use specific names (correct)
mixology menu add-drink --menu-id mnu-abc --drink-id drk-xyz
mixology order place --menu-id mnu-abc drk-abc:2

# Inventory uses ingredient-id (correct - keyed by ingredient, not PK)
mixology inventory get --ingredient-id ing-abc
```

## Tasks

- [x] Update `menu.go` show command: `--menu-id` → `--id`
- [x] Update `menu.go` publish command: `--menu-id` → `--id`
- [x] Update `orders.go` get command: `--order-id` → `--id`
- [x] Update `orders.go` complete command: `--order-id` → `--id`
- [x] Update `orders.go` cancel command: `--order-id` → `--id`
- [x] Verify CLI commands work with new flag names

## Acceptance Criteria

- [x] Single-entity PK operations use `--id`
- [x] Multi-entity operations use entity-specific names (`--menu-id`, `--drink-id`)
- [x] Inventory commands keep `--ingredient-id` (lookup key, not PK)
- [x] All CLI commands work correctly
