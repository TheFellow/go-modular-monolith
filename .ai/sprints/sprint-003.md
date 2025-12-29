# Sprint 003: CLI Skeleton + List Command

## Goal

Wire up the CLI entry point and expose the list command.

## Tasks

- [ ] Initialize go.mod with urfave/cli/v3 dependency
- [ ] Create `main/cli/main.go` with CLI skeleton
- [ ] Create `app/drinks/drinks.go` module surface exposing use cases
- [ ] Create `app/app.go` composition root (minimal, no middleware yet)
- [ ] Create `app/drinks_accessor.go` DrinksAccessor with List method
- [ ] Wire `list` subcommand to DrinksAccessor.List

## Success Criteria

- `go run ./main/cli list` prints seeded drinks
- `go build ./...` passes

## Dependencies

- Sprint 002 (List query, seed data)
