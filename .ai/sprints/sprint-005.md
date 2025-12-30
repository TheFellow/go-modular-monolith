# Sprint 005: Middleware Infrastructure

## Goal

Build the middleware chain infrastructure with stub implementations. Query and Command pipelines have distinct type signatures.

## Tasks

- [x] Create `pkg/middleware/context.go` with Context, WithPrincipal, etc.
- [x] Create `pkg/middleware/query.go` with QueryChain, QueryMiddleware, QueryNext types
- [x] Create `pkg/middleware/command.go` with CommandChain, CommandMiddleware, CommandNext types
- [x] Create `pkg/middleware/run.go` with RunQuery and RunCommand functions
- [x] Create package-level chains: `middleware.Query` and `middleware.Command`
- [x] Create `pkg/dispatcher/dispatcher.go` with no-op Dispatcher stub
- [x] Create `pkg/uow/uow.go` with UnitOfWork Manager temp impl
- [x] Create stub middleware: `QueryAuthZ` (pass-through), `CommandAuthZ`, `UnitOfWork`, `Dispatcher`
- [x] Update drinks accessor to use `middleware.RunQuery`

## Notes

- **Query pipeline**: Takes `cedar.EntityUID` as resource (just identity for authz)
- **Command pipeline**: Takes `cedar.Entity` as resource (full entity with attributes)

This split makes the semantics explicit - queries only need to identify what's being read, commands need the full entity for policy evaluation and mutation.

## Success Criteria

- `go test ./...` passes
- `go run ./main/cli list` still works through middleware chain

## Dependencies

- Sprint 004 (Get query, module surface pattern)
