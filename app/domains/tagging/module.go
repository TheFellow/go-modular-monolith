package tagging

import (
	"sort"

	taggingauthz "github.com/TheFellow/go-modular-monolith/app/domains/tagging/authz"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

type discoveryResult[T any] struct {
	value  T
	entity cedar.Entity
}

func (r discoveryResult[T]) CedarEntity() cedar.Entity { return r.entity }

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

// Replace makes desired the target's complete tag set. Adding or changing
// values requires the owning domain's tag action; removing keys requires its
// untag action. Mixed changes require both actions against the before and
// after states. The replacement is recorded as one stable tag activity;
// untag is an additional authorization requirement rather than a second
// activity.
func (m *Module) Replace(ctx *middleware.Context, target cedar.EntityUID, desired tag.Tags) (Result, error) {
	registration, err := m.resolve(target)
	if err != nil {
		return Result{}, err
	}
	if err := desired.Validate(); err != nil {
		return Result{}, err
	}
	desired = desired.Sorted()

	return middleware.RunCommand(m.pipeline, ctx, middleware.CommandSpec[targetState, Result]{
		Action: registration.TagAction,
		AuthorizationActions: func(current targetState) []cedar.EntityUID {
			return replaceActions(registration, current.tags, desired)
		},
		Load: func(ctx *middleware.Context) (targetState, error) {
			return loadState(ctx, registration, target)
		},
		Handle: func(ctx *middleware.Context, _ targetState) (Result, error) {
			changed, err := m.repository.Replace(ctx, target, desired)
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

// Show returns active entity references matching an exact tag, or every value
// of its key when exact is false. Authorization belongs solely to the tagging
// domain and is not repeated against the referenced entities.
func (m *Module) Show(ctx *middleware.Context, value tag.Tag, exact bool) ([]Reference, error) {
	if err := value.Validate(); err != nil {
		return nil, err
	}
	resource := taggingauthz.TagDiscovery{
		UID: cedar.NewEntityUID(taggingauthz.TagDiscoveryType, "show"),
		Key: value.Key, Value: value.Value, Exact: exact,
	}
	result, err := middleware.RunEntityQuery(m.pipeline, ctx, taggingauthz.ActionShow,
		func(queryCtx store.Context, _ struct{}) (discoveryResult[[]Reference], error) {
			associations, err := m.repository.find(queryCtx, value, exact)
			if err != nil {
				return discoveryResult[[]Reference]{}, err
			}
			active, err := m.activeAssociations(queryCtx, associations)
			if err != nil {
				return discoveryResult[[]Reference]{}, err
			}
			refs := make([]Reference, 0, len(active))
			for _, association := range active {
				refs = append(refs, Reference{
					EntityType: entityTypeName(association.target.Type),
					EntityID:   string(association.target.ID),
					Tag:        association.tag.String(),
				})
			}
			return discoveryResult[[]Reference]{value: refs, entity: resource.CedarEntity()}, nil
		}, struct{}{})
	if err != nil {
		return nil, err
	}
	return result.value, nil
}

// Summary aggregates active associations by canonical tag.
func (m *Module) Summary(ctx *middleware.Context) ([]Summary, error) {
	resource := taggingauthz.TagDiscovery{
		UID: cedar.NewEntityUID(taggingauthz.TagDiscoveryType, "summary"),
	}
	result, err := middleware.RunEntityQuery(m.pipeline, ctx, taggingauthz.ActionSummary,
		func(queryCtx store.Context, _ struct{}) (discoveryResult[[]Summary], error) {
			associations, err := m.repository.all(queryCtx)
			if err != nil {
				return discoveryResult[[]Summary]{}, err
			}
			active, err := m.activeAssociations(queryCtx, associations)
			if err != nil {
				return discoveryResult[[]Summary]{}, err
			}
			byTag := make(map[string]*Summary)
			for _, association := range active {
				canonical := association.tag.String()
				row := byTag[canonical]
				if row == nil {
					row = &Summary{Tag: canonical}
					byTag[canonical] = row
				}
				row.Total++
				switch association.target.Type {
				case entity.TypeDrink:
					row.Drinks++
				case entity.TypeIngredient:
					row.Ingredients++
				case entity.TypeInventory:
					row.Inventory++
				case entity.TypeMenu:
					row.Menus++
				case entity.TypeOrder:
					row.Orders++
				}
			}
			rows := make([]Summary, 0, len(byTag))
			for _, row := range byTag {
				rows = append(rows, *row)
			}
			sort.Slice(rows, func(i, j int) bool {
				if rows[i].Total != rows[j].Total {
					return rows[i].Total > rows[j].Total
				}
				return rows[i].Tag < rows[j].Tag
			})
			return discoveryResult[[]Summary]{value: rows, entity: resource.CedarEntity()}, nil
		}, struct{}{})
	if err != nil {
		return nil, err
	}
	return result.value, nil
}

func (m *Module) activeAssociations(ctx store.Context, associations []association) ([]association, error) {
	idsByType := make(map[cedar.EntityType][]cedar.String)
	seen := make(map[cedar.EntityUID]struct{})
	for _, association := range associations {
		if _, ok := seen[association.target]; ok {
			continue
		}
		seen[association.target] = struct{}{}
		idsByType[association.target.Type] = append(idsByType[association.target.Type], association.target.ID)
	}
	activeByType := make(map[cedar.EntityType]map[cedar.String]struct{}, len(idsByType))
	for entityType, ids := range idsByType {
		registration, err := m.registry.resolve(entityType)
		if err != nil {
			return nil, err
		}
		active, err := registration.Active(ctx, ids)
		if err != nil {
			return nil, err
		}
		activeByType[entityType] = active
	}
	result := make([]association, 0, len(associations))
	for _, association := range associations {
		if _, active := activeByType[association.target.Type][association.target.ID]; active {
			result = append(result, association)
		}
	}
	return result, nil
}

func entityTypeName(value cedar.EntityType) string {
	switch value {
	case entity.TypeDrink:
		return "Drink"
	case entity.TypeIngredient:
		return "Ingredient"
	case entity.TypeInventory:
		return "Inventory"
	case entity.TypeMenu:
		return "Menu"
	case entity.TypeOrder:
		return "Order"
	default:
		return string(value)
	}
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

func replaceActions(registration Target, current, desired tag.Tags) []cedar.EntityUID {
	currentByKey := current.Map()
	desiredByKey := desired.Map()
	requiresTag := false
	requiresUntag := false
	for key, value := range desiredByKey {
		if existing, ok := currentByKey[key]; !ok || existing != value {
			requiresTag = true
		}
	}
	for key := range currentByKey {
		if _, ok := desiredByKey[key]; !ok {
			requiresUntag = true
		}
	}

	actions := make([]cedar.EntityUID, 0, 2)
	if requiresTag || !requiresUntag {
		actions = append(actions, registration.TagAction)
	}
	if requiresUntag {
		actions = append(actions, registration.UntagAction)
	}
	return actions
}
