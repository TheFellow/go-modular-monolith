# Sprint 021: Drinks — Delete + Filtered List

## Goal

1. Add a first-class “delete drink” capability in the Drinks domain.
2. Add filtered listing for drinks, primarily to find a drink by name.
3. Ensure Menu reacts to deleted drinks (removes them from menus) within the existing atomic, in-transaction event dispatch model.

## Problem

- There is no way to remove a drink once created, so menus and other read models can accumulate stale items.
- `Drinks.List` is all-or-nothing; users commonly need “find by name” or other lightweight filtering without a dedicated query endpoint.
- We need to ensure cross-context consistency: when a drink is deleted, any menu entries referencing it should be removed automatically.

## Solution

### Delete Drink

- Add a `delete` command in the Drinks module that:
  1. Validates the drink ID.
  2. Loads the existing drink (for NotFound and event payload).
  3. Deletes the record inside the command transaction.
  4. Emits a `DrinkDeleted` domain event (at minimum includes `DrinkID`, optionally includes `Name`).
- Add authz for delete (`ActionDelete`) and policy (owner-only).

### Menu Reaction

- Add a Menu handler for `events.DrinkDeleted` that:
  - Scans menus and removes items referencing the deleted `DrinkID`.
  - Runs in the same transaction as the delete command (existing event dispatch behavior), so the delete + menu cleanup are atomic.
- Update dispatcher codegen so the new handler is actually wired.

### Filtered List (Find by Name)

- Extend `Drinks.List` request to accept an optional filter (start with `Name`).
- Implement filtering in the DAO using bstore queries (the repo uses `pkg/store` + `github.com/mjl-/bstore`, not “dstore”):
  - `DrinkRow.Name` is `bstore:"unique"` already, so `FilterEqual("Name", name)` should be efficient and index-backed for exact matches.

## Bstore Query Pattern (the “dstore” piece)

The DAO runs queries either:
- inside an existing transaction from `store.TxFromContext(ctx)` (commands/handlers), or
- in a read tx via `store.FromContext(ctx).Read(...)` (queries).

Exact name match (leverages `DrinkRow.Name` unique index):

```go
rows, err := bstore.QueryTx[DrinkRow](tx).
    FilterEqual("Name", filter.Name).
    SortAsc("Name").
    List()
```

Notes:
- `FilterEqual` is the primary mechanism for “find by name”.
- If we later want substring search (`contains`), bstore doesn’t provide SQL LIKE; we can add a `FilterFn` fallback (in-memory) or introduce a derived/indexed field (e.g., normalized `NameLower`) for scalable prefix/contains search.

## API / Surface Changes

- `app/domains/drinks/authz/actions.go`
  - Add `ActionDelete`.
- `app/domains/drinks/authz/policies.cedar`
  - Permit delete for owner principals.
- `app/domains/drinks/events`
  - Add `DrinkDeleted`.
- `app/domains/drinks`
  - Add `Delete` module method (shape TBD; likely `DeleteRequest{ID}` returning deleted `models.Drink` for confirmation).
  - Extend `ListRequest` with optional filter fields (start with `Name`).
- `main/cli/drinks.go`
  - Add `drinks delete <id>`.
  - Add list flag `--name <exact>` (or `--filter-name`) wired into `ListRequest`.

## Persistence Changes

- `app/domains/drinks/internal/dao`
  - Add `Delete(ctx, id)` using `tx.Delete(&DrinkRow{ID: ...})`.
  - Update `List(ctx, filter)` to use bstore query filters when requested.

## Menu Handler

- `app/domains/menu/handlers`
  - Add `DrinkDeletedMenuUpdater.Handle(ctx, events.DrinkDeleted)`:
    - `menuDAO.List(ctx)`
    - For each menu, filter out items whose `DrinkID` matches.
    - If changed, `menuDAO.Update(ctx, menu)`.

## Dispatcher Wiring

- Add/adjust handler(s) and run codegen:
  - `go generate ./pkg/dispatcher`
  - Verify `pkg/dispatcher/dispatcher_gen.go` includes `DrinkDeleted` case mapping to the new menu handler.

## Tests

- Drinks
  - Add a command test verifying:
    - delete removes drink
    - delete of missing ID returns NotFound
    - delete emits `DrinkDeleted` event
  - Add a query/DAO test verifying `List` filter by name returns only matching drink.
- Dispatcher/Menu integration
  - Add a `pkg/dispatcher` integration test similar to `TestDispatch_StockAdjusted_UpdatesMenuAvailability`:
    - create menu with a drink item
    - dispatch `DrinkDeleted{DrinkID: ...}`
    - assert menu no longer contains that drink item

## Tasks

- [x] Add `ActionDelete` + Cedar policy for owner-only delete
- [x] Add `events.DrinkDeleted`
- [x] Implement drinks delete command + module surface + CLI command
- [x] Implement `DAO.Delete` and list filtering using bstore (`FilterEqual("Name", ...)`)
- [x] Add Menu handler for `DrinkDeleted`
- [x] Run dispatcher codegen and update generated file
- [x] Add tests (drinks delete, list filter, dispatcher/menu integration)
- [x] Verify `go test ./...` passes

## Acceptance Criteria

- `drinks delete <id>` removes a drink and returns a clear success output; deleting a missing drink returns NotFound.
- `drinks list --name Margarita` returns only matching drink(s).
- Any menu containing the deleted drink is updated (item removed) in the same atomic transaction as the delete command.
- `go generate ./pkg/dispatcher` and `go test ./...` pass with deterministic output.

