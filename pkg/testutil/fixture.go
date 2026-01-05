package testutil

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu"
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
	Ctx     *middleware.Context
	Metrics *telemetry.MemoryMetrics

	Drinks      *drinks.Module
	Ingredients *ingredients.Module
	Inventory   *inventory.Module
	Menu        *menu.Module
	Orders      *orders.Module
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
	ctx := a.Context(context.Background(), p)

	return &Fixture{
		T:       t,
		Store:   s,
		App:     a,
		Ctx:     ctx,
		Metrics: metrics,

		Drinks:      a.Drinks,
		Ingredients: a.Ingredients,
		Inventory:   a.Inventory,
		Menu:        a.Menu,
		Orders:      a.Orders,
	}
}

func (f *Fixture) AsActor(actor string) *middleware.Context {
	f.T.Helper()
	p, err := authn.ParseActor(actor)
	Ok(f.T, err)
	return f.App.Context(context.Background(), p)
}

func (f *Fixture) Bootstrap() *Bootstrap {
	f.T.Helper()
	return &Bootstrap{fix: f}
}
