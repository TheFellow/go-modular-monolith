package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (d *FileMenuDAO) Update(ctx context.Context, menu Menu) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := d.ensureLoaded(ctx); err != nil {
		return err
	}
	if menu.ID == "" {
		return errors.Invalidf("menu id is required")
	}
	if menu.Name == "" {
		return errors.Invalidf("menu name is required")
	}

	for i, existing := range d.menus {
		if existing.ID == menu.ID {
			d.menus[i] = menu
			return nil
		}
	}

	return errors.NotFoundf("menu %s not found", menu.ID)
}
