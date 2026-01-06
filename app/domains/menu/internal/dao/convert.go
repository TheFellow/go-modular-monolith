package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func toRow(m menumodels.Menu) MenuRow {
	items := make([]MenuItemRow, 0, len(m.Items))
	for _, it := range m.Items {
		var price optional.Value[money.Price]
		if p, ok := it.Price.Unwrap(); ok {
			price = optional.Some(money.Price(p))
		} else {
			price = optional.None[money.Price]()
		}

		items = append(items, MenuItemRow{
			DrinkID:      string(it.DrinkID.ID),
			DisplayName:  it.DisplayName,
			Price:        price,
			Featured:     it.Featured,
			Availability: string(it.Availability),
			SortOrder:    it.SortOrder,
		})
	}

	return MenuRow{
		ID:          string(m.ID.ID),
		Name:        m.Name,
		Description: m.Description,
		Items:       items,
		Status:      string(m.Status),
		CreatedAt:   m.CreatedAt,
		PublishedAt: m.PublishedAt,
	}
}

func toModel(r MenuRow) menumodels.Menu {
	items := make([]menumodels.MenuItem, 0, len(r.Items))
	for _, it := range r.Items {
		var price optional.Value[menumodels.Price]
		if p, ok := it.Price.Unwrap(); ok {
			price = optional.Some(menumodels.Price(money.Price(p)))
		} else {
			price = optional.None[menumodels.Price]()
		}

		items = append(items, menumodels.MenuItem{
			DrinkID:      models.NewDrinkID(it.DrinkID),
			DisplayName:  it.DisplayName,
			Price:        price,
			Featured:     it.Featured,
			Availability: menumodels.Availability(it.Availability),
			SortOrder:    it.SortOrder,
		})
	}

	return menumodels.Menu{
		ID:          menumodels.NewMenuID(r.ID),
		Name:        r.Name,
		Description: r.Description,
		Items:       items,
		Status:      menumodels.MenuStatus(r.Status),
		CreatedAt:   r.CreatedAt,
		PublishedAt: r.PublishedAt,
	}
}
