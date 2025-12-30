# Mixology as a Service - Progress Summary

## Current Status

**Phase:** Planning Complete
**Next Sprint:** TBD

## What We're Building

A modular monolith demonstrating DDD/CQRS patterns with Cedar-based authorization. The domain is "Mixology as a Service" - cocktail/drink management.

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Entry point | `/main/cli` (future: `/main/server`) | Multiple surfaces share same app |
| Module location | `/app/*` | Bounded contexts as top-level packages |
| Infrastructure | `/pkg/*` | Non-domain supporting code |
| Auth | Cedar policies via `cedar-go` | Fine-grained, declarative authz |
| Persistence | File-based JSON (initially) | Quick start, interface-driven for swap |
| CLI framework | urfave/cli v3 | Simple, well-documented |

## Key Patterns

- **Two pipelines:** Query (read) vs Command (write) with different middleware
- **Use cases own authz:** Action + Resource defined on use case struct
- **App facade:** `app/app.go` composes all dependencies, exposes fluent accessors
- **No event cascading:** Handlers cannot produce additional domain events

## Sprint Progress

| Sprint | Description | Status |
|--------|-------------|--------|
| 001 | Catalog read model + file DAO | Completed |
| 002 | Seed data + list query | Completed |
| 003 | CLI skeleton + list command | Completed |
| 004 | Get query | Completed |
| 005 | Middleware infrastructure | Completed |
| 006 | First write use case + AuthZ | Completed |
| 007 | Uniform error handling | Completed |

## Open Items

- [x] Drink data format (ID + Name)
- [ ] Testing approach (table-driven vs acceptance)

## Recent Changes

- Implemented minimal `Drink` domain model + file-backed DAO.
- Implemented `List` query + unit test.
- Added CLI `list` command wired to app facade/accessor.
- Added `Get` query + CLI `get` command.
- Added middleware chains + stubs (AuthZ/UoW/Dispatcher) and routed reads through `middleware.Query`.
- Added Cedar policy codegen + real AuthZ middleware + `create` command (denies anonymous).
- Added `pkg/errors` codegen + updated app to return `Invalid`/`NotFound`/`Internal` errors.
