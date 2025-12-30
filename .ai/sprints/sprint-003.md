# Sprint 003: CLI Skeleton + List Command

## Goal

Wire up the CLI entry point and expose the list command.

## Tasks

- [x] Add urfave/cli/v3 dependency
- [x] Create `main/cli/main.go` with CLI skeleton
- [x] Create `app/app.go` facade that instantiates and exposes modules
- [x] Create `app/drinks/module.go` drinks Module that owns use cases
- [x] Wire `list` subcommand to `drinks.Module.List`

## Notes

Modules own their use cases directly and pull middleware from package-level `middleware.Query` and `middleware.Command` chains. Initially we skip middleware entirely - it's added in Sprint 005.

## Success Criteria

- `go run ./main/cli list` prints drinks
- `go build ./...` passes

## Dependencies

- Sprint 002 (List query)
