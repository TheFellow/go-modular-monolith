# Sprint 001: Drinks Read Model + File DAO

## Goal

Establish the foundational drinks module with a Drink model and file-based persistence.

## Tasks

- [ ] Create `app/drinks/models/drink.go` with Drink struct (domain model)
- [ ] Create `app/drinks/internal/dao/drink.go` with file-based DrinkDAO
- [ ] Create `pkg/data/drinks.json` with initial seed data (pending data format)

## Notes

Request/response types live in the module root as the public API:
- `app/drinks/get.go` defines `GetRequest` and `GetResponse`
- `app/drinks/list.go` defines `ListRequest` and `ListResponse`
- `app/drinks/create.go` defines `CreateRequest` and `CreateResponse`

The module root delegates to internal implementations, transforming requests as needed. For example, `Module.Get(ctx, GetRequest)` calls `queries.Get.Execute(ctx, req.ID)`. The internal implementations have whatever signature makes sense for them.

## Success Criteria

- `go build ./...` passes
- DrinkDAO can read/write drinks to JSON file

## Dependencies

- None

## Open Items

- Drink data format: What fields does your seed data include?
