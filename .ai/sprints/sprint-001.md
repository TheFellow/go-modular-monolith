# Sprint 001: Drinks Read Model + File DAO

## Goal

Establish the foundational drinks module with a Drink model and file-based persistence.

## Tasks

- [x] Create `app/drinks/models/drink.go` with Drink struct (domain model, no JSON tags)
- [x] Create `app/drinks/internal/dao/drink.go` with persistence Drink record model (JSON tags + `deleted_at`)
- [x] Create `app/drinks/internal/dao/dao.go` with file-based `FileDrinkDAO`
- [x] Create `pkg/data/drinks.json` with initial drink data (ID + Name)

## Notes

Request/response types live in the module root as the public API:
- `app/drinks/get.go` defines `GetRequest` and `GetResponse`
- `app/drinks/list.go` defines `ListRequest` and `ListResponse`
- `app/drinks/create.go` defines `CreateRequest` and `CreateResponse`

The module root delegates to internal implementations, transforming requests as needed. For example, `Module.Get(ctx, GetRequest)` calls `queries.Get.Execute(ctx, req.ID)`. The internal implementations have whatever signature makes sense for them.

## Success Criteria

- `go build ./...` passes
- DAO can read/write drinks to JSON file

## Dependencies

- None

## Open Items

- None
