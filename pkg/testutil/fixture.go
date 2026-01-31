package testutil

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
)

type Fixture struct {
	T       testing.TB
	Store   *store.Store
	App     *app.App
	Metrics *telemetry.MemoryMetrics

	Audit       *audit.Module
	Drinks      *drinks.Module
	Ingredients *ingredients.Module
	Inventory   *inventory.Module
	Menu        *menus.Module
	Orders      *orders.Module

	ownerCtx *middleware.Context
}

func NewFixture(t testing.TB) *Fixture {
	t.Helper()

	path := filepath.Join(t.TempDir(), "mixology.test.db")
	s, err := store.Open(path)
	Ok(t, err)

	metrics := telemetry.Memory()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	a := app.New(
		app.WithStore(s),
		app.WithLogger(logger),
		app.WithMetrics(metrics),
	)
	t.Cleanup(func() { _ = a.Close() })

	p, err := authn.ParseActor("owner")
	Ok(t, err)
	ownerCtx := a.Context(context.Background(), p)

	return &Fixture{
		T:       t,
		Store:   s,
		App:     a,
		Metrics: metrics,

		Audit:       a.Audit,
		Drinks:      a.Drinks,
		Ingredients: a.Ingredients,
		Inventory:   a.Inventory,
		Menu:        a.Menu,
		Orders:      a.Orders,

		ownerCtx: ownerCtx,
	}
}

func (f *Fixture) OwnerContext() *middleware.Context {
	f.T.Helper()
	return f.ownerCtx
}

func (f *Fixture) ActorContext(actor string) *middleware.Context {
	f.T.Helper()
	p, err := authn.ParseActor(actor)
	Ok(f.T, err)
	return f.App.Context(context.Background(), p)
}

func (f *Fixture) Bootstrap() *Bootstrap {
	f.T.Helper()
	return &Bootstrap{fix: f}
}
