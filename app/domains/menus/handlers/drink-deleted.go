package handlers

import (
	"fmt"
	events "github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"strings"
)

type DrinkDeleted struct{ dao *dao.DAO }

func NewDrinkDeleted(s *store.Store, tags tag.Repository) *DrinkDeleted {
	return &DrinkDeleted{dao: dao.New(s, tags)}
}
func (h *DrinkDeleted) Handling(ctx *middleware.HandlerContext, e events.DrinkDeleted) error {
	menus, err := h.dao.ListByDrink(ctx, e.Drink.ID)
	if err != nil {
		return err
	}
	if len(menus) == 0 {
		return nil
	}
	refs := make([]string, 0, len(menus))
	for _, menu := range menus {
		ctx.ReferenceEntity(menu.ID.EntityUID())
		refs = append(refs, fmt.Sprintf("%s (%s, %s)", menu.Name, menu.ID.String(), menu.Status))
	}
	return errors.FailedPreconditionf("cannot delete drink %q: used by menus %s; return published menus to draft and remove the drink first", e.Drink.Name, strings.Join(refs, ", "))
}
func (h *DrinkDeleted) Handle(*middleware.HandlerContext, events.DrinkDeleted) error { return nil }
