package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (d *FileMenuDAO) Get(ctx context.Context, id string) (Menu, bool, error) {
	if err := ctx.Err(); err != nil {
		return Menu{}, false, err
	}
	if err := d.ensureLoaded(ctx); err != nil {
		return Menu{}, false, err
	}
	if id == "" {
		return Menu{}, false, errors.Invalidf("id is required")
	}

	for _, m := range d.menus {
		if m.ID == id {
			return m, true, nil
		}
	}
	return Menu{}, false, nil
}
