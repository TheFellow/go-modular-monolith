# Sprint 001: Drinks Read Model + File DAO

## Goal

Establish the foundational drinks module with a Drink model and file-based persistence.

## Tasks

- [x] Create `app/drinks/models/drink.go` with Drink struct (domain model)
- [x] Create `app/drinks/internal/dao/drink.go` with persistence `dao.Drink` record (JSON serialization)
- [x] Create `app/drinks/internal/dao/dao.go` with file-based `dao.FileDrinkDAO`
- [x] Create `app/drinks/get.go` with GetRequest/GetResponse
- [x] Create `app/drinks/list.go` with ListRequest/ListResponse
- [x] Create `app/drinks/create.go` with CreateRequest/CreateResponse

## Notes

Request/response types live in the module root as the public API. The module root delegates to internal implementations, transforming requests as needed.

## Success Criteria

- `go build ./...` passes
- DrinkDAO can read/write drinks to JSON file

## Dependencies

- None
