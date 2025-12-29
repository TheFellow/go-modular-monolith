# Sprint 002: Seed Data + List Query

## Goal

Load seed data and implement the List query use case.

## Tasks

- [ ] Populate `pkg/data/drinks.json` with drink seed data
- [ ] Create `app/drinks/authz/actions.go` with action EntityUIDs
- [ ] Create `app/drinks/queries/list.go` with List use case struct
- [ ] Write unit test that loads and parses seed data

## Success Criteria

- Unit test loads and parses seed data successfully
- `go test ./...` passes

## Dependencies

- Sprint 001 (Drink model, DAO)
