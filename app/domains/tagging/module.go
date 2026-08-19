package tagging

import (
	"sort"
	"strings"

	taggingauthz "github.com/TheFellow/go-modular-monolith/app/domains/tagging/authz"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/set"
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
	name   string
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
	return m.pipeline.LoadCommand(ctx, registration.TagAction,
		func(ctx *middleware.Context) (targetState, error) {
			return loadState(ctx, registration, target)
		},
		func(ctx *middleware.Context, _ targetState) (Result, error) {
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
	)
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

	return m.pipeline.LoadCommandActions(ctx, registration.TagAction,
		func(ctx *middleware.Context) (targetState, error) {
			return loadState(ctx, registration, target)
		},
		func(ctx *middleware.Context, _ targetState) (Result, error) {
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
		func(current targetState) []cedar.EntityUID {
			return replaceActions(registration, current.tags, desired)
		},
	)
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
	return m.pipeline.LoadCommand(ctx, registration.UntagAction,
		func(ctx *middleware.Context) (targetState, error) {
			return loadState(ctx, registration, target)
		},
		func(ctx *middleware.Context, _ targetState) (Result, error) {
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
	)
}

// List returns tags only after authorizing the owning domain's read action
// against the target's complete current state.
func (m *Module) List(ctx *middleware.Context, target cedar.EntityUID) (tag.Tags, error) {
	registration, err := m.resolve(target)
	if err != nil {
		return nil, err
	}
	state, err := m.pipeline.Query(ctx, registration.GetAction, target,
		func(queryCtx store.Context, _ cedar.EntityUID) (targetState, error) {
			return loadState(queryCtx, registration, target)
		})
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
	return m.pipeline.QueryResource(ctx, taggingauthz.ActionShow, resource.CedarEntity(), value,
		func(queryCtx store.Context, _ tag.Tag) ([]Reference, error) {
			associations, err := m.repository.find(queryCtx, value, exact)
			if err != nil {
				return nil, err
			}
			active, err := m.activeAssociations(queryCtx, associations)
			if err != nil {
				return nil, err
			}
			refs := make([]Reference, 0, len(active))
			names := make(map[cedar.EntityUID]string)
			for _, association := range active {
				name, ok := names[association.target]
				if !ok {
					registration, resolveErr := m.registry.resolve(association.target.Type)
					if resolveErr != nil {
						return nil, resolveErr
					}
					state, loadErr := loadState(queryCtx, registration, association.target)
					if loadErr != nil {
						return nil, loadErr
					}
					name = state.name
					names[association.target] = name
				}
				refs = append(refs, Reference{
					EntityType: entityTypeName(association.target.Type),
					EntityName: name,
					EntityID:   string(association.target.ID),
					Tag:        association.tag.String(),
				})
			}
			return refs, nil
		})
}

// Summary aggregates active associations by canonical tag.
func (m *Module) Summary(ctx *middleware.Context) ([]Summary, error) {
	resource := taggingauthz.TagDiscovery{
		UID: cedar.NewEntityUID(taggingauthz.TagDiscoveryType, "summary"),
	}
	return m.pipeline.QueryResource(ctx, taggingauthz.ActionSummary, resource.CedarEntity(), struct{}{},
		func(queryCtx store.Context, _ struct{}) ([]Summary, error) {
			associations, err := m.repository.all(queryCtx)
			if err != nil {
				return nil, err
			}
			active, err := m.activeAssociations(queryCtx, associations)
			if err != nil {
				return nil, err
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
			return rows, nil
		})
}

func (m *Module) activeAssociations(ctx store.Context, associations []association) ([]association, error) {
	idsByType := make(map[cedar.EntityType][]cedar.String)
	var seen set.Set[cedar.EntityUID]
	for _, association := range associations {
		if seen.Contains(association.target) {
			continue
		}
		seen.Add(association.target)
		idsByType[association.target.Type] = append(idsByType[association.target.Type], association.target.ID)
	}
	activeByType := make(map[cedar.EntityType]set.Set[cedar.String], len(idsByType))
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
		if activeByType[association.target.Type].Contains(association.target.ID) {
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
	name := strings.TrimSpace(loaded.DisplayName)
	if name == "" {
		return targetState{}, errors.Internalf("tag target loader returned an empty display name for %s", target)
	}
	return targetState{target: target, entity: loaded.Entity, name: name, tags: loaded.Tags}, nil
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
