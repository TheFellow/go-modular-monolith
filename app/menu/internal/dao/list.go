package dao

import "context"

func (d *FileMenuDAO) List(ctx context.Context) ([]Menu, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := d.ensureLoaded(ctx); err != nil {
		return nil, err
	}

	out := make([]Menu, len(d.menus))
	copy(out, d.menus)
	return out, nil
}
