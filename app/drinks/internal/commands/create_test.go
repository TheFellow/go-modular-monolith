package commands_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/uow"
)

func TestCreate_PersistsOnCommit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "drinks.json")

	err := os.WriteFile(path, []byte("[]\n"), 0o644)
	testutil.ErrorIf(t, err != nil, "write seed: %v", err)

	d := dao.NewFileDrinkDAO(path)
	err = d.Load(context.Background())
	testutil.ErrorIf(t, err != nil, "load: %v", err)

	ctx := middleware.NewContext(context.Background())
	tx, err := uow.NewManager().Begin(ctx)
	testutil.ErrorIf(t, err != nil, "begin tx: %v", err)
	ctx = middleware.NewContext(ctx, middleware.WithUnitOfWork(tx))

	uc := commands.NewCreate(d)
	created, err := uc.Execute(ctx, "Margarita")
	testutil.ErrorIf(t, err != nil, "execute: %v", err)
	testutil.ErrorIf(t, string(created.ID.ID) == "", "expected id to be set")

	err = tx.Commit()
	testutil.ErrorIf(t, err != nil, "commit: %v", err)

	loaded := dao.NewFileDrinkDAO(path)
	err = loaded.Load(context.Background())
	testutil.ErrorIf(t, err != nil, "reload: %v", err)

	drinks, err := loaded.List(context.Background())
	testutil.ErrorIf(t, err != nil, "list: %v", err)

	testutil.ErrorIf(t, len(drinks) != 1, "expected 1 drink, got %d", len(drinks))
	testutil.ErrorIf(t, drinks[0].Name != "Margarita", "expected Margarita, got %q", drinks[0].Name)
}
