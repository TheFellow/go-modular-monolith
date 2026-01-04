package dao

import (
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/menu/models"
)

func FromDomain(m models.Menu) Menu {
	items := make([]MenuItem, 0, len(m.Items))
	for _, item := range m.Items {
		items = append(items, MenuItem{
			DrinkID:      string(item.DrinkID.ID),
			DisplayName:  item.DisplayName,
			Price:        fromDomainPrice(item.Price),
			Featured:     item.Featured,
			Availability: string(item.Availability),
			SortOrder:    item.SortOrder,
		})
	}

	return Menu{
		ID:          string(m.ID.ID),
		Name:        m.Name,
		Description: m.Description,
		Items:       items,
		Status:      string(m.Status),
		CreatedAt:   m.CreatedAt,
		PublishedAt: m.PublishedAt,
	}
}

func (m Menu) ToDomain() models.Menu {
	items := make([]models.MenuItem, 0, len(m.Items))
	for _, item := range m.Items {
		items = append(items, models.MenuItem{
			DrinkID:      drinksmodels.NewDrinkID(item.DrinkID),
			DisplayName:  item.DisplayName,
			Price:        item.Price.toDomain(),
			Featured:     item.Featured,
			Availability: models.Availability(item.Availability),
			SortOrder:    item.SortOrder,
		})
	}

	return models.Menu{
		ID:          models.NewMenuID(m.ID),
		Name:        m.Name,
		Description: m.Description,
		Items:       items,
		Status:      models.MenuStatus(m.Status),
		CreatedAt:   m.CreatedAt,
		PublishedAt: m.PublishedAt,
	}
}

func fromDomainPrice(p *models.Price) *Price {
	if p == nil {
		return nil
	}
	return &Price{Amount: p.Amount, Currency: p.Currency}
}

func (p *Price) toDomain() *models.Price {
	if p == nil {
		return nil
	}
	return &models.Price{Amount: p.Amount, Currency: p.Currency}
}
