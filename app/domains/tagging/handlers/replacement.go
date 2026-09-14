package handlers

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

type replacement struct {
	repository *dao.Repository
	change     tag.Replacement
	before     tag.Tags
}

func newReplacement(s *store.Store) *replacement {
	return &replacement{repository: dao.NewRepository(s)}
}
func (h *replacement) prepare(ctx *middleware.HandlerContext, change tag.Replacement) error {
	if change.Edit.Desired == nil {
		return errors.Invalidf("tag replacement is required")
	}
	if err := change.Edit.Desired.Validate(); err != nil {
		return err
	}
	before, err := h.repository.List(ctx, change.Entity.UID)
	if err != nil {
		return err
	}
	if change.Edit.Expected != nil && before.Canonical().String() != change.Edit.Expected.Canonical().String() {
		return errors.Conflictf("tags changed; reload before replacing the complete set")
	}
	desired := *change.Edit.Desired
	add, remove := false, false
	for key, value := range desired.Map() {
		if old, ok := before.Map()[key]; !ok || old != value {
			add = true
		}
	}
	for key := range before.Map() {
		if _, ok := desired.Map()[key]; !ok {
			remove = true
		}
	}
	actions := []cedar.EntityUID{}
	if add || !remove {
		actions = append(actions, change.TagAction)
	}
	if remove {
		actions = append(actions, change.UntagAction)
	}
	for _, values := range []tag.Tags{before, desired} {
		resource := change.Entity
		attrs := cedar.RecordMap{}
		for key, value := range values.Map() {
			attrs[cedar.String(key)] = cedar.String(value)
		}
		resource.Tags = cedar.NewRecord(attrs)
		for _, action := range actions {
			if err := authz.AuthorizeWithEntity(ctx.Principal(), action, resource); err != nil {
				return err
			}
		}
	}
	h.change, h.before = change, before
	return nil
}
func (h *replacement) apply(ctx *middleware.HandlerContext) error {
	changed, err := h.repository.Replace(ctx, h.change.Entity.UID, *h.change.Edit.Desired)
	if err != nil {
		return err
	}
	if changed {
		ctx.RecordEffect("tags_changed", h.change.Entity.UID, middleware.Change("tags", h.before.Canonical().String(), h.change.Edit.Desired.Canonical().String()))
	}
	return nil
}
