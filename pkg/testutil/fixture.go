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
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
)

type Fixture struct {
	T       testing.TB
	Store   *store.Store
	App     *app.Session
	Metrics *telemetry.MemoryMetrics

	Audit       *audit.Module
	Drinks      *drinks.Module
	Ingredients *ingredients.Module
	Inventory   *inventory.Module
	Menus       *menus.Module
	Orders      *orders.Module

	ownerCtx *middleware.Context
	ctx      context.Context
	closed   bool
}

func NewFixture(t testing.TB) *Fixture {
	t.Helper()

	path := filepath.Join(t.TempDir(), "mixology.test.db")
	metrics := telemetry.Memory()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := log.ToContext(context.Background(), logger)
	ctx = telemetry.WithMetrics(ctx, metrics)
	p, err := authn.ParseActor("owner")
	Ok(t, err)
	ctx = authn.ToContext(ctx, p)
	s, err := store.Open(ctx, path)
	Ok(t, err)
	application := app.New(ctx, app.Config{Store: s})
	ownerCtx := middleware.NewContext(ctx)
	a := app.NewSession(ownerCtx, application)

	f := &Fixture{
		T:       t,
		Store:   s,
		App:     a,
		Metrics: metrics,

		Audit:       a.Audit,
		Drinks:      a.Drinks,
		Ingredients: a.Ingredients,
		Inventory:   a.Inventory,
		Menus:       a.Menus,
		Orders:      a.Orders,

		ownerCtx: ownerCtx,
		ctx:      ctx,
	}
	t.Cleanup(func() { Ok(t, f.Close()) })
	return f
}

func (f *Fixture) OwnerContext() *middleware.Context {
	f.T.Helper()
	return f.ownerCtx
}

func (f *Fixture) ActorContext(actor string) *middleware.Context {
	f.T.Helper()
	p, err := authn.ParseActor(actor)
	Ok(f.T, err)
	return middleware.NewContext(authn.ToContext(f.ctx, p))
}

func (f *Fixture) Bootstrap() *Bootstrap {
	f.T.Helper()
	return &Bootstrap{fix: f}
}

func (f *Fixture) Close() error {
	if f.closed {
		return nil
	}
	f.closed = true
	return f.App.Close()
}
