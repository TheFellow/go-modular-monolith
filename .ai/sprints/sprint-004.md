# Sprint 004: Get Query

## Goal

Implement the Get query use case for fetching a single drink by ID.

## Tasks

- [ ] Create `app/drinks/queries/get.go` with Get use case struct
- [ ] Update `app/drinks_accessor.go` with Get method and use case
- [ ] Add `get` subcommand to CLI
- [ ] Write unit test for Get query

## Success Criteria

- `go run ./main/cli get <id>` prints drink details
- `go test ./...` passes

## Dependencies

- Sprint 003 (CLI skeleton, accessor pattern)
