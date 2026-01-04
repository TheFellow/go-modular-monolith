package dao

import (
	"time"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func FromDomain(m models.Menu) Menu {
	var publishedAt *time.Time
	if t, ok := m.PublishedAt.Unwrap(); ok {
		publishedAt = &t
	}

	items := make([]MenuItem, 0, len(m.Items))
	for _, item := range m.Items {
		var displayName *string
		if s, ok := item.DisplayName.Unwrap(); ok {
			displayName = &s
		}

		items = append(items, MenuItem{
			DrinkID:      string(item.DrinkID.ID),
			DisplayName:  displayName,
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
		PublishedAt: publishedAt,
	}
}

func (m Menu) ToDomain() models.Menu {
	var publishedAt optional.Value[time.Time] = optional.None[time.Time]()
	if m.PublishedAt != nil {
		publishedAt = optional.Some(*m.PublishedAt)
	}

	items := make([]models.MenuItem, 0, len(m.Items))
	for _, item := range m.Items {
		var displayName optional.Value[string] = optional.None[string]()
		if item.DisplayName != nil {
			displayName = optional.Some(*item.DisplayName)
		}

		items = append(items, models.MenuItem{
			DrinkID:      drinksmodels.NewDrinkID(item.DrinkID),
			DisplayName:  displayName,
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
		PublishedAt: publishedAt,
	}
}

func fromDomainPrice(p optional.Value[models.Price]) *Price {
	v, ok := p.Unwrap()
	if !ok {
		return nil
	}
	return &Price{Amount: v.Amount, Currency: v.Currency}
}

func (p *Price) toDomain() optional.Value[models.Price] {
	if p == nil {
		return optional.None[models.Price]()
	}
	return optional.Some(models.Price{Amount: p.Amount, Currency: p.Currency})
}
