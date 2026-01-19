# Sprint 038: Role-Based Access Control

## Goal

Remove redundant owner policies and implement proper role-based access control with specialized actors.

## Problem

Every domain has policies like:

```cedar
permit(
    principal == Mixology::Actor::"owner",
    action == Mixology::Ingredient::Action::"create",
    resource
);
```

But `pkg/authz/base.cedar` already grants owner full access:

```cedar
permit(
    principal == Mixology::Actor::"owner",
    action,
    resource
);
```

These domain-specific owner policies are redundant and clutter the policy files.

## Redundant Policies to Remove

### ingredients/authz/policies.cedar
- `owner` + `create` (lines 17-21)
- `owner` + `update` (lines 23-27)
- `owner` + `delete` (lines 29-33)

### drinks/authz/policies.cedar
- `owner` + `create` (lines 17-21)
- `owner` + `update` (lines 36-40)
- `owner` + `delete` (lines 51-55)

### menu/authz/policies.cedar
- `owner` + `create` (lines 15-19)
- `owner` + `add_drink` (lines 21-25)
- `owner` + `remove_drink` (lines 27-31)
- `owner` + `publish` (lines 33-37)

### orders/authz/policies.cedar
- `owner` + `place/complete/cancel` (lines 12-20)

### inventory/authz/policies.cedar
- `owner` + `adjust/set` (lines 3-10)

## New Role Definitions

### Actors

| Actor | Description |
|-------|-------------|
| `owner` | Full access to everything (base policy) |
| `manager` | Operations lead - can manage menus, inventory, and all drinks |
| `sommelier` | Wine specialist - manages wine drinks |
| `bartender` | Bar staff - manages non-wine drinks, handles orders |
| `anonymous` | Unauthenticated - read-only access |

### Permission Matrix

| Domain | Action | owner | manager | sommelier | bartender | anonymous |
|--------|--------|-------|---------|-----------|-----------|-----------|
| **Audit** | list, get | ✓ | ✗ | ✗ | ✗ | ✗ |
| **Drinks** | list, get | ✓ | ✓ | ✓ | ✓ | ✓ |
| **Drinks** | create, update, delete (wine) | ✓ | ✓ | ✓ | ✗ | ✗ |
| **Drinks** | create, update, delete (non-wine) | ✓ | ✓ | ✗ | ✓ | ✗ |
| **Ingredients** | list, get | ✓ | ✓ | ✓ | ✓ | ✓ |
| **Ingredients** | create, update, delete | ✓ | ✓ | ✗ | ✗ | ✗ |
| **Inventory** | list, get | ✓ | ✓ | ✓ | ✓ | ✓ |
| **Inventory** | adjust, set | ✓ | ✓ | ✗ | ✗ | ✗ |
| **Menu** | list, get | ✓ | ✓ | ✓ | ✓ | ✓ |
| **Menu** | create, add_drink, remove_drink, publish | ✓ | ✓ | ✗ | ✗ | ✗ |
| **Orders** | list, get | ✓ | ✓ | ✓ | ✓ | ✗ |
| **Orders** | place, complete, cancel | ✓ | ✓ | ✗ | ✓ | ✗ |

## Updated Policies

### pkg/authz/base.cedar (unchanged)

```cedar
// Owner can do anything.
permit(
    principal == Mixology::Actor::"owner",
    action,
    resource
);

// Allow anonymous users to login.
permit(
    principal == Mixology::Actor::"anonymous",
    action == Mixology::Action::"login",
    resource == Mixology::Auth::Session::"current"
);
```

### app/domains/audit/authz/policies.cedar (new)

```cedar
// Audit logs are owner-only.
// No additional policies needed - base policy covers owner.
// Explicitly forbid other actors from audit access.

forbid(
    principal in [
        Mixology::Actor::"manager",
        Mixology::Actor::"sommelier",
        Mixology::Actor::"bartender",
        Mixology::Actor::"anonymous"
    ],
    action,
    resource is Mixology::Activity
);
```

### app/domains/drinks/authz/policies.cedar

```cedar
// Anyone can read drinks.
permit(
    principal,
    action in [
        Mixology::Drink::Action::"list",
        Mixology::Drink::Action::"get"
    ],
    resource is Mixology::Drink
);

// Managers can manage all drinks.
permit(
    principal == Mixology::Actor::"manager",
    action in [
        Mixology::Drink::Action::"create",
        Mixology::Drink::Action::"update",
        Mixology::Drink::Action::"delete"
    ],
    resource is Mixology::Drink
);

// Sommeliers can manage wine drinks.
permit(
    principal == Mixology::Actor::"sommelier",
    action in [
        Mixology::Drink::Action::"create",
        Mixology::Drink::Action::"update",
        Mixology::Drink::Action::"delete"
    ],
    resource is Mixology::Drink
) when {
    resource.category == "wine"
};

// Bartenders can manage non-wine drinks.
permit(
    principal == Mixology::Actor::"bartender",
    action in [
        Mixology::Drink::Action::"create",
        Mixology::Drink::Action::"update",
        Mixology::Drink::Action::"delete"
    ],
    resource is Mixology::Drink
) when {
    resource.category != "wine"
};
```

### app/domains/ingredients/authz/policies.cedar

```cedar
// Anyone can read ingredients.
permit(
    principal,
    action in [
        Mixology::Ingredient::Action::"list",
        Mixology::Ingredient::Action::"get"
    ],
    resource is Mixology::Ingredient
);

// Managers can modify ingredients.
permit(
    principal == Mixology::Actor::"manager",
    action in [
        Mixology::Ingredient::Action::"create",
        Mixology::Ingredient::Action::"update",
        Mixology::Ingredient::Action::"delete"
    ],
    resource is Mixology::Ingredient
);
```

### app/domains/inventory/authz/policies.cedar

```cedar
// Staff can read inventory.
permit(
    principal in [
        Mixology::Actor::"manager",
        Mixology::Actor::"sommelier",
        Mixology::Actor::"bartender"
    ],
    action in [
        Mixology::Inventory::Action::"list",
        Mixology::Inventory::Action::"get"
    ],
    resource is Mixology::Inventory
);

// Managers can modify inventory.
permit(
    principal == Mixology::Actor::"manager",
    action in [
        Mixology::Inventory::Action::"adjust",
        Mixology::Inventory::Action::"set"
    ],
    resource is Mixology::Inventory
);
```

### app/domains/menu/authz/policies.cedar

```cedar
// Anyone can read menus.
permit(
    principal,
    action in [
        Mixology::Menu::Action::"list",
        Mixology::Menu::Action::"get"
    ],
    resource is Mixology::Menu
);

// Managers can manage menus.
permit(
    principal == Mixology::Actor::"manager",
    action in [
        Mixology::Menu::Action::"create",
        Mixology::Menu::Action::"add_drink",
        Mixology::Menu::Action::"remove_drink",
        Mixology::Menu::Action::"publish"
    ],
    resource is Mixology::Menu
);
```

### app/domains/orders/authz/policies.cedar

```cedar
// Staff can read orders.
permit(
    principal in [
        Mixology::Actor::"manager",
        Mixology::Actor::"sommelier",
        Mixology::Actor::"bartender"
    ],
    action in [
        Mixology::Order::Action::"list",
        Mixology::Order::Action::"get"
    ],
    resource is Mixology::Order
);

// Bartenders and Managers can manage orders.
permit(
    principal in [
        Mixology::Actor::"manager",
        Mixology::Actor::"bartender"
    ],
    action in [
        Mixology::Order::Action::"place",
        Mixology::Order::Action::"complete",
        Mixology::Order::Action::"cancel"
    ],
    resource is Mixology::Order
);
```

## Actor Registration

### pkg/authn/actor.go

```go
package authn

import cedar "github.com/cedar-policy/cedar-go"

const TypeActor = cedar.EntityType("Mixology::Actor")

var (
    Owner     = cedar.NewEntityUID(TypeActor, "owner")
    Manager   = cedar.NewEntityUID(TypeActor, "manager")
    Sommelier = cedar.NewEntityUID(TypeActor, "sommelier")
    Bartender = cedar.NewEntityUID(TypeActor, "bartender")
    Anonymous = cedar.NewEntityUID(TypeActor, "anonymous")
)

// ParseActor returns an actor EntityUID from a string.
func ParseActor(s string) (cedar.EntityUID, error) {
    switch s {
    case "owner":
        return Owner, nil
    case "manager":
        return Manager, nil
    case "sommelier":
        return Sommelier, nil
    case "bartender":
        return Bartender, nil
    case "anonymous":
        return Anonymous, nil
    default:
        return cedar.EntityUID{}, errors.Invalidf("unknown actor: %s", s)
    }
}
```

## Tasks

### Phase 1: Remove Redundant Policies

- [x] Remove owner policies from `ingredients/authz/policies.cedar`
- [x] Remove owner policies from `drinks/authz/policies.cedar`
- [x] Remove owner policies from `menu/authz/policies.cedar`
- [x] Remove owner policies from `orders/authz/policies.cedar`
- [x] Remove owner policies from `inventory/authz/policies.cedar`

### Phase 2: Update Actor Registration

- [x] Add `Manager`, `Sommelier`, and `Bartender` to `pkg/authn/authn.go`
- [x] Update `ParseActor` to handle new actors

### Phase 3: Update Domain Policies

- [x] Create `audit/authz/policies.cedar` (owner-only)
- [x] Update `drinks/authz/policies.cedar` (manager=all, sommelier=wine, bartender=non-wine)
- [x] Update `ingredients/authz/policies.cedar` (manager=cud, staff=read)
- [x] Update `inventory/authz/policies.cedar` (manager=cud, staff=read)
- [x] Update `menu/authz/policies.cedar` (manager=cud, others=read)
- [x] Update `orders/authz/policies.cedar` (manager/bartender=manage)

### Phase 4: Update CLI

- [x] Add `--actor` flag to CLI for testing different roles
- [x] Default to `owner` for backward compatibility

### Phase 5: Verify

- [x] Test owner has full access
- [x] Test manager has broad operational access but not audit
- [x] Test sommelier can only manage wine drinks
- [x] Test bartender can manage non-wine drinks and orders
- [x] Test anonymous has read-only access
- [x] Test audit is owner-only
- [x] Run all tests

## Acceptance Criteria

- [x] No redundant owner policies in domain files
- [x] Base policy is the single source of owner permissions
- [x] Manager can perform all operational tasks (CUD drinks/ingredients/menus/inventory)
- [x] Sommelier can CUD wine drinks only
- [x] Bartender can CUD non-wine drinks only
- [x] Bartender and Manager can place/complete/cancel orders
- [x] Only owner can access audit logs
- [x] All tests pass

## Result

```bash
# Manager - broad access
$ mixology --actor manager drinks create ...       # ✓
$ mixology --actor manager inventory adjust ...    # ✓
$ mixology --actor manager audit list              # ✗ Forbidden

# Sommelier - wine only
$ mixology --actor sommelier drinks create --category wine ...     # ✓
$ mixology --actor sommelier drinks create --category cocktail ... # ✗ Forbidden

# Bartender - non-wine
$ mixology --actor bartender drinks create --category cocktail ... # ✓
$ mixology --actor bartender drinks create --category wine ...     # ✗ Forbidden

# Anonymous - read only
$ mixology --actor anonymous drinks list    # ✓
$ mixology --actor anonymous drinks create  # ✗ Forbidden
```
