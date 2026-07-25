package tagging

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

// Result is the current tag state after a mutation.
type Result struct {
	Target  cedar.EntityUID
	Tags    tag.Tags
	Changed bool
	entity  cedar.Entity
}

func (r Result) CedarEntity() cedar.Entity { return r.entity }

type targetState struct {
	target cedar.EntityUID
	entity cedar.Entity
	tags   tag.Tags
}

func (s targetState) CedarEntity() cedar.Entity { return s.entity }

// Module provides authorized tag operations across registered operational
// domains. Persistence remains private so callers cannot bypass policy.
type Module struct {
	repository *Repository
	registry   *Registry
	pipeline   *middleware.Pipeline
}

func NewModule(repository *Repository, registry *Registry, pipeline *middleware.Pipeline) *Module {
	return &Module{repository: repository, registry: registry, pipeline: pipeline}
}

// Upsert adds value or replaces the existing value for its key.
func (m *Module) Upsert(ctx *middleware.Context, target cedar.EntityUID, value tag.Tag) (Result, error) {
	registration, err := m.resolve(target)
	if err != nil {
		return Result{}, err
	}
	if err := value.Validate(); err != nil {
		return Result{}, err
	}
	return middleware.RunCommand(m.pipeline, ctx, middleware.CommandSpec[targetState, Result]{
		Action: registration.TagAction,
		Load: func(ctx *middleware.Context) (targetState, error) {
			return loadState(ctx, registration, target)
		},
		Handle: func(ctx *middleware.Context, _ targetState) (Result, error) {
			changed, err := m.repository.Upsert(ctx, target, value)
			if err != nil {
				return Result{}, err
			}
			state, err := loadState(ctx, registration, target)
			if err != nil {
				return Result{}, err
			}
			if changed {
				ctx.TouchEntity(target)
			}
			return resultFromState(state, changed), nil
		},
	})
}

// Set is an alias for Upsert for clients that describe key replacement as a
// set operation.
func (m *Module) Set(ctx *middleware.Context, target cedar.EntityUID, value tag.Tag) (Result, error) {
	return m.Upsert(ctx, target, value)
}

// Remove deletes the tag identified by key. A missing key is a successful
// no-op.
func (m *Module) Remove(ctx *middleware.Context, target cedar.EntityUID, key string) (Result, error) {
	registration, err := m.resolve(target)
	if err != nil {
		return Result{}, err
	}
	if err := (tag.Tag{Key: key}).Validate(); err != nil {
		return Result{}, err
	}
	return middleware.RunCommand(m.pipeline, ctx, middleware.CommandSpec[targetState, Result]{
		Action: registration.UntagAction,
		Load: func(ctx *middleware.Context) (targetState, error) {
			return loadState(ctx, registration, target)
		},
		Handle: func(ctx *middleware.Context, _ targetState) (Result, error) {
			changed, err := m.repository.Remove(ctx, target, key)
			if err != nil {
				return Result{}, err
			}
			state, err := loadState(ctx, registration, target)
			if err != nil {
				return Result{}, err
			}
			if changed {
				ctx.TouchEntity(target)
			}
			return resultFromState(state, changed), nil
		},
	})
}

// List returns tags only after authorizing the owning domain's read action
// against the target's complete current state.
func (m *Module) List(ctx *middleware.Context, target cedar.EntityUID) (tag.Tags, error) {
	registration, err := m.resolve(target)
	if err != nil {
		return nil, err
	}
	state, err := middleware.RunEntityQuery(m.pipeline, ctx, registration.GetAction,
		func(queryCtx store.Context, _ cedar.EntityUID) (targetState, error) {
			return loadState(queryCtx, registration, target)
		}, target)
	if err != nil {
		return nil, err
	}
	return state.tags, nil
}

func (m *Module) resolve(target cedar.EntityUID) (Target, error) {
	if err := validateTarget(target); err != nil {
		return Target{}, err
	}
	return m.registry.resolve(target.Type)
}

func loadState(ctx store.Context, registration Target, target cedar.EntityUID) (targetState, error) {
	loaded, err := registration.Load(ctx, target.ID)
	if err != nil {
		return targetState{}, err
	}
	if loaded.Entity.UID != target {
		return targetState{}, errors.Internalf("tag target loader returned %s for %s", loaded.Entity.UID, target)
	}
	return targetState{target: target, entity: loaded.Entity, tags: loaded.Tags}, nil
}

func resultFromState(state targetState, changed bool) Result {
	return Result{Target: state.target, Tags: state.tags, Changed: changed, entity: state.entity}
}
