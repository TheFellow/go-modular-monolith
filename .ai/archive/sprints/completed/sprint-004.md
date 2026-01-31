# Sprint 004: Get Query

## Goal

Implement the Get query use case for fetching a single drink by ID.

## Tasks

- [x] Create `app/drinks/queries/get.go` with `Queries.Get(...)`
- [x] Update `app/drinks/module.go` with `Get(...)` method
- [x] Add `get` subcommand to CLI
- [x] Write unit test for Get query

## Success Criteria

- `go run ./main/cli get <id>` prints drink details
- `go test ./...` passes

## Dependencies

- Sprint 003 (CLI skeleton, module surface pattern)
