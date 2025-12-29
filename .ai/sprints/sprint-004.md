# Sprint 004: Seed Command (Idempotent)

## Goal

Add CLI command to seed/reset drink data from the bundled seed file.

## Tasks

- [ ] Add `seed` subcommand to CLI
- [ ] Implement idempotent seed logic (overwrites existing data)
- [ ] Ensure seed data path is configurable or uses sensible default

## Success Criteria

- `go run ./main/cli seed` loads drinks from `pkg/data/drinks.json`
- Running `seed` twice produces the same result
- `go run ./main/cli list` shows seeded drinks after seed

## Dependencies

- Sprint 003 (CLI skeleton, list command)
