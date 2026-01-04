package store

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/mjl-/bstore"
)

var (
	mu sync.Mutex
	DB *bstore.DB
)

type txKey struct{}

func WithTx(ctx context.Context, tx *bstore.Tx) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, txKey{}, tx)
}

func TxFromContext(ctx context.Context) (*bstore.Tx, bool) {
	if ctx == nil {
		return nil, false
	}
	tx, ok := ctx.Value(txKey{}).(*bstore.Tx)
	return tx, ok
}

func Open(path string) error {
	mu.Lock()
	defer mu.Unlock()

	if DB != nil {
		return errors.Internalf("store already open")
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return errors.Internalf("mkdir db dir: %w", err)
		}
	}

	db, err := bstore.Open(context.Background(), path, nil,
		drinksmodels.Drink{},
		ingredientsmodels.Ingredient{},
		inventorymodels.Stock{},
		menumodels.Menu{},
		ordersmodels.Order{},
	)
	if err != nil {
		return err
	}
	DB = db
	return nil
}

func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if DB == nil {
		return nil
	}
	err := DB.Close()
	DB = nil
	return err
}
