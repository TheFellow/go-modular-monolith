# Sprint 006: First Write Use Case + AuthZ

## Goal

Implement CreateDrink command with full authz infrastructure.

## Tasks

- [x] Add cedar-go dependency
- [x] Create `pkg/authz/base.cedar` with anonymous login policy
- [x] Create `app/drinks/authz/policies.cedar` with drinks policies
- [x] Create `pkg/authz/authorize.go` with Authorize functions
- [x] Create policy embedding codegen (`policies_gen.go` via `go generate ./pkg/authz`)
- [x] Create `app/drinks/internal/commands/create.go` with Create use case
- [x] Create `app/drinks/events/drink_created.go` domain event
- [x] Update `app/drinks/module.go` with Create use case and method (using `middleware.RunCommand`)
- [x] Add `create` subcommand to CLI
- [x] Implement real AuthZ middleware (replace stub)
- [x] Create `pkg/authn/authn.go` with fake principal helpers

## Success Criteria

- `go run ./main/cli create "Margarita"` works with owner principal
- AuthZ denies anonymous users from creating drinks
- `go test ./...` passes

## Dependencies

- Sprint 005 (middleware infrastructure)
