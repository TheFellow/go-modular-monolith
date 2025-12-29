# Sprint 005: Middleware Infrastructure

## Goal

Build the middleware chain infrastructure without authz (stub).

## Tasks

- [ ] Create `pkg/middleware/context.go` with Context, WithPrincipal, etc.
- [ ] Create `pkg/middleware/middleware.go` with Chain, Middleware, Next types
- [ ] Create `pkg/middleware/run.go` with generic Run function
- [ ] Create `pkg/dispatcher/dispatcher.go` with no-op Dispatcher stub
- [ ] Create `pkg/uow/uow.go` with no-op UnitOfWork Manager stub
- [ ] Create stub middleware: `authz.go` (pass-through), `uow.go`, `dispatcher.go`
- [ ] Update `app/app.go` to compose query and command chains
- [ ] Update `app/drinks.go` to use middleware.Run for List

## Success Criteria

- `go test ./...` passes
- `go run ./main/cli list` still works through middleware chain

## Dependencies

- Sprint 003 (app.go, drinks.go)
