package models

import (
	"strings"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

const MenuEntityType = cedar.EntityType("Mixology::Menu")

func NewMenuID(id string) cedar.EntityUID {
	return cedar.NewEntityUID(MenuEntityType, cedar.String(id))
}

type Menu struct {
	ID          cedar.EntityUID
	Name        string
	Description string
	Items       []MenuItem
	Status      MenuStatus
	CreatedAt   time.Time
	PublishedAt *time.Time
}

func (m Menu) EntityUID() cedar.EntityUID {
	return m.ID
}

func (m Menu) Validate() error {
	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" {
		return errors.Invalidf("name is required")
	}
	if err := m.Status.Validate(); err != nil {
		return err
	}
	for i, item := range m.Items {
		if err := item.Validate(); err != nil {
			return errors.Invalidf("item %d: %w", i, err)
		}
	}
	return nil
}

type MenuItem struct {
	DrinkID      cedar.EntityUID
	DisplayName  string
	Price        *Price
	Featured     bool
	Availability Availability
	SortOrder    int
}

func (i MenuItem) Validate() error {
	if string(i.DrinkID.ID) == "" {
		return errors.Invalidf("drink id is required")
	}
	if err := i.Availability.Validate(); err != nil {
		return err
	}
	if i.Price != nil {
		if err := i.Price.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type Availability string

const (
	AvailabilityAvailable   Availability = "available"
	AvailabilityLimited     Availability = "limited"
	AvailabilityUnavailable Availability = "unavailable"
)

func (a Availability) Validate() error {
	switch a {
	case AvailabilityAvailable, AvailabilityLimited, AvailabilityUnavailable:
		return nil
	default:
		return errors.Invalidf("invalid availability %q", string(a))
	}
}

type MenuStatus string

const (
	MenuStatusDraft     MenuStatus = "draft"
	MenuStatusPublished MenuStatus = "published"
	MenuStatusArchived  MenuStatus = "archived"
)

func (s MenuStatus) Validate() error {
	switch s {
	case MenuStatusDraft, MenuStatusPublished, MenuStatusArchived:
		return nil
	default:
		return errors.Invalidf("invalid status %q", string(s))
	}
}

type Price struct {
	Amount   int
	Currency string
}

func (p Price) Validate() error {
	if p.Amount < 0 {
		return errors.Invalidf("amount must be >= 0")
	}
	if strings.TrimSpace(p.Currency) == "" {
		return errors.Invalidf("currency is required")
	}
	return nil
}
