package tagging

import (
	"sync"

	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/set"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

// LoadTarget loads an operational entity through its owning domain's private
// query path. The returned value must contain the entity's complete Cedar
// state, including its current tags.
type LoadTarget func(store.Context, cedar.String) (TargetState, error)

// ActiveTargets returns the subset of ids whose entities are currently
// active. Implementations query their owning domain in bulk and do not
// perform authorization.
type ActiveTargets func(store.Context, []cedar.String) (set.Set[cedar.String], error)

// TargetState is the complete domain-owned state needed by tag orchestration.
type TargetState struct {
	Entity cedar.Entity
	Tags   tag.Tags
}

// Target describes the domain-owned authorization and loading behavior for a
// taggable entity type.
type Target struct {
	Type        cedar.EntityType
	GetAction   cedar.EntityUID
	TagAction   cedar.EntityUID
	UntagAction cedar.EntityUID
	Load        LoadTarget
	Active      ActiveTargets
}

// Registry lets operational domains register tag behavior without making the
// tagging domain depend on their models or private persistence packages.
type Registry struct {
	mu      sync.RWMutex
	targets map[cedar.EntityType]Target
}

func NewRegistry() *Registry {
	return &Registry{targets: make(map[cedar.EntityType]Target)}
}

// Register adds one operational target type. Duplicate and incomplete
// registrations are programmer errors and panic during application assembly.
func (r *Registry) Register(target Target) {
	if r == nil || target.Type == "" || target.GetAction.IsZero() || target.TagAction.IsZero() || target.UntagAction.IsZero() || target.Load == nil || target.Active == nil {
		panic("tagging: incomplete target registration")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.targets[target.Type]; ok {
		panic("tagging: duplicate target registration for " + string(target.Type))
	}
	r.targets[target.Type] = target
}

func (r *Registry) resolve(entityType cedar.EntityType) (Target, error) {
	if r == nil {
		return Target{}, errors.Internalf("tag target registry is required")
	}
	r.mu.RLock()
	target, ok := r.targets[entityType]
	r.mu.RUnlock()
	if !ok {
		return Target{}, errors.Invalidf("unsupported tag target type: %s", entityType)
	}
	return target, nil
}
