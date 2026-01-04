package models

import (
	"strings"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
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
	PublishedAt optional.Value[time.Time]
}

func (m Menu) EntityUID() cedar.EntityUID {
	return m.ID
}

func (m Menu) CedarEntity() cedar.Entity {
	uid := m.ID
	if string(uid.ID) == "" {
		uid = cedar.NewEntityUID(MenuEntityType, cedar.String(""))
	}
	return cedar.Entity{
		UID:        uid,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
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
	DisplayName  optional.Value[string]
	Price        optional.Value[Price]
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
	if p, ok := i.Price.Unwrap(); ok {
		if err := p.Validate(); err != nil {
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

type Price = money.Price
