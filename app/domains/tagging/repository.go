// Package tagging persists user-authored tags for entities across application domains.
package tagging

import (
	"context"
	"errors"

	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

// Repository stores tags independently of the entity's owning domain.
type Repository struct {
	store *store.Store
}

// NewRepository returns a tag repository backed by s.
func NewRepository(s *store.Store) *Repository {
	return &Repository{store: s}
}

// RegisterSchema registers the polymorphic tag association with s.
func RegisterSchema(ctx context.Context, s *store.Store) {
	s.Register(ctx, entityTagRow{})
}

// List returns the target's tags in key order.
func (r *Repository) List(ctx store.Context, target cedar.EntityUID) (tag.Tags, error) {
	if err := validateTarget(target); err != nil {
		return nil, err
	}

	var tagsByTarget map[cedar.EntityUID]tag.Tags
	err := r.store.ReadContext(ctx, func(tx *bstore.Tx) error {
		var err error
		tagsByTarget, err = r.ListTypeTx(tx, target.Type, []cedar.String{target.ID})
		return err
	})
	if err != nil {
		return nil, store.MapError(err, "list tags for %s", target)
	}
	return tagsByTarget[target], nil
}

// ListTypeTx reads tags for the requested entities with one type-scoped query.
// The returned map only contains targets that have tags. Callers can use this
// inside the read transaction that loaded their domain entities.
func (r *Repository) ListTypeTx(tx *bstore.Tx, entityType cedar.EntityType, ids []cedar.String) (map[cedar.EntityUID]tag.Tags, error) {
	if tx == nil {
		return nil, apperrors.Internalf("tag read transaction is required")
	}
	if len(ids) == 0 {
		return map[cedar.EntityUID]tag.Tags{}, nil
	}

	values := make([]any, len(ids))
	for i, id := range ids {
		target := cedar.NewEntityUID(entityType, id)
		if err := validateTarget(target); err != nil {
			return nil, err
		}
		values[i] = string(id)
	}

	rows, err := bstore.QueryTx[entityTagRow](tx).
		FilterEqual("EntityType", string(entityType)).
		FilterEqual("EntityID", values...).
		SortAsc("EntityID", "Key").
		List()
	if err != nil {
		return nil, err
	}

	result := make(map[cedar.EntityUID]tag.Tags)
	for _, row := range rows {
		target := cedar.NewEntityUID(cedar.EntityType(row.EntityType), cedar.String(row.EntityID))
		result[target] = append(result[target], tag.Tag{Key: row.Key, Value: row.Value})
	}
	return result, nil
}

// Upsert adds a tag or replaces the value for its key. It reports whether the
// persisted association changed.
func (r *Repository) Upsert(ctx store.Context, target cedar.EntityUID, value tag.Tag) (bool, error) {
	if err := validateTarget(target); err != nil {
		return false, err
	}
	if err := value.Validate(); err != nil {
		return false, err
	}

	changed := false
	err := store.Write(ctx, func(tx *bstore.Tx) error {
		row, err := findRow(tx, target, value.Key)
		if errors.Is(err, bstore.ErrAbsent) {
			row = entityTagRow{
				EntityType: string(target.Type),
				EntityID:   string(target.ID),
				Key:        value.Key,
				Value:      value.Value,
			}
			if err := tx.Insert(&row); err != nil {
				return err
			}
			changed = true
			return nil
		}
		if err != nil {
			return err
		}
		if row.Value == value.Value {
			return nil
		}
		row.Value = value.Value
		if err := tx.Update(&row); err != nil {
			return err
		}
		changed = true
		return nil
	})
	if err != nil {
		return false, store.MapError(err, "upsert tag %q for %s", value.Key, target)
	}
	return changed, nil
}

// Replace makes desired the target's complete tag set. It reports whether any
// association was added, updated, or removed.
func (r *Repository) Replace(ctx store.Context, target cedar.EntityUID, desired tag.Tags) (bool, error) {
	if err := validateTarget(target); err != nil {
		return false, err
	}
	if err := desired.Validate(); err != nil {
		return false, err
	}
	desired = desired.Sorted()

	changed := false
	err := store.Write(ctx, func(tx *bstore.Tx) error {
		rows, err := bstore.QueryTx[entityTagRow](tx).
			FilterEqual("EntityType", string(target.Type)).
			FilterEqual("EntityID", string(target.ID)).
			List()
		if err != nil {
			return err
		}

		remaining := make(map[string]entityTagRow, len(rows))
		for _, row := range rows {
			remaining[row.Key] = row
		}
		for _, value := range desired {
			row, exists := remaining[value.Key]
			if !exists {
				row = entityTagRow{
					EntityType: string(target.Type),
					EntityID:   string(target.ID),
					Key:        value.Key,
					Value:      value.Value,
				}
				if err := tx.Insert(&row); err != nil {
					return err
				}
				changed = true
				continue
			}
			delete(remaining, value.Key)
			if row.Value == value.Value {
				continue
			}
			row.Value = value.Value
			if err := tx.Update(&row); err != nil {
				return err
			}
			changed = true
		}
		for _, row := range remaining {
			if err := tx.Delete(&row); err != nil {
				return err
			}
			changed = true
		}
		return nil
	})
	if err != nil {
		return false, store.MapError(err, "replace tags for %s", target)
	}
	return changed, nil
}

// Remove deletes the target's tag with key. It reports whether an association
// was removed; a missing key is a successful no-op.
func (r *Repository) Remove(ctx store.Context, target cedar.EntityUID, key string) (bool, error) {
	if err := validateTarget(target); err != nil {
		return false, err
	}
	if err := (tag.Tag{Key: key}).Validate(); err != nil {
		return false, err
	}

	changed := false
	err := store.Write(ctx, func(tx *bstore.Tx) error {
		row, err := findRow(tx, target, key)
		if errors.Is(err, bstore.ErrAbsent) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := tx.Delete(&row); err != nil {
			return err
		}
		changed = true
		return nil
	})
	if err != nil {
		return false, store.MapError(err, "remove tag %q from %s", key, target)
	}
	return changed, nil
}

// DeleteTarget removes every tag association for target and returns the count.
func (r *Repository) DeleteTarget(ctx store.Context, target cedar.EntityUID) (int, error) {
	if err := validateTarget(target); err != nil {
		return 0, err
	}

	deleted := 0
	err := store.Write(ctx, func(tx *bstore.Tx) error {
		var err error
		deleted, err = bstore.QueryTx[entityTagRow](tx).
			FilterEqual("EntityType", string(target.Type)).
			FilterEqual("EntityID", string(target.ID)).
			Delete()
		return err
	})
	if err != nil {
		return 0, store.MapError(err, "delete tags for %s", target)
	}
	return deleted, nil
}

func findRow(tx *bstore.Tx, target cedar.EntityUID, key string) (entityTagRow, error) {
	return bstore.QueryTx[entityTagRow](tx).
		FilterEqual("EntityType", string(target.Type)).
		FilterEqual("EntityID", string(target.ID)).
		FilterEqual("Key", key).
		Get()
}

func validateTarget(target cedar.EntityUID) error {
	id := string(target.ID)
	var err error
	switch target.Type {
	case entity.TypeDrink:
		_, err = entity.ParseDrinkID(id)
	case entity.TypeIngredient:
		_, err = entity.ParseIngredientID(id)
	case entity.TypeMenu:
		_, err = entity.ParseMenuID(id)
	case entity.TypeOrder:
		_, err = entity.ParseOrderID(id)
	case entity.TypeInventory:
		_, err = entity.ParseInventoryID(id)
	default:
		return apperrors.Invalidf("unsupported tag target type: %s", target.Type)
	}
	if err != nil {
		return apperrors.Invalidf("invalid tag target %s: %v", target, err)
	}
	return nil
}
