# Sprint 006: First Write Use Case + AuthZ

## Goal

Implement CreateDrink command with full authz infrastructure.

## Tasks

- [ ] Add cedar-go dependency
- [ ] Create `pkg/authz/base.cedar` with anonymous login policy
- [ ] Create `app/drinks/authz/policies.cedar` with drinks policies
- [ ] Create `pkg/authz/authorize.go` with Authorize function
- [ ] Create policy embedding codegen (`policies_gen.go`)
- [ ] Create `app/drinks/internal/commands/create.go` with Create use case
- [ ] Create `app/drinks/events/drink_created.go` domain event
- [ ] Update `app/drinks_accessor.go` with Create use case and method (using middleware.Command)
- [ ] Add `create` subcommand to CLI
- [ ] Implement real AuthZ middleware (replace stub)
- [ ] Create `pkg/authn/authn.go` with fake principal helpers

## Success Criteria

- `go run ./main/cli create "Margarita"` works with owner principal
- AuthZ denies anonymous users from creating drinks
- `go test ./...` passes

## Dependencies

- Sprint 005 (middleware infrastructure)
