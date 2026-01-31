package cli

import (
	"encoding/json"
	"io"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type OrderRow struct {
	ID        string `table:"ID" json:"id"`
	MenuID    string `table:"MENU_ID" json:"menu_id"`
	Status    string `table:"STATUS" json:"status"`
	CreatedAt string `table:"CREATED_AT" json:"created_at"`
}

type OrderDetail struct {
	ID          string `table:"-" json:"id"`
	MenuID      string `table:"-" json:"menu_id"`
	Status      string `table:"-" json:"status"`
	CreatedAt   string `table:"-" json:"created_at"`
	CompletedAt string `table:"-" json:"completed_at,omitempty"`
	Notes       string `table:"-" json:"notes,omitempty"`
}

type OrderItemRow struct {
	DrinkID  string `table:"DRINK_ID" json:"drink_id"`
	Quantity int    `table:"QUANTITY" json:"quantity"`
}

type OrderInput struct {
	MenuID string         `json:"menu_id"`
	Items  []OrderItemRow `json:"items"`
}

func ToOrderRow(o *models.Order) OrderRow {
	if o == nil {
		return OrderRow{}
	}
	return OrderRow{
		ID:        o.ID.String(),
		MenuID:    o.MenuID.String(),
		Status:    string(o.Status),
		CreatedAt: formatTime(o.CreatedAt),
	}
}

func ToOrderRows(items []*models.Order) []OrderRow {
	rows := make([]OrderRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, ToOrderRow(item))
	}
	return rows
}

func ToOrderDetail(o *models.Order) OrderDetail {
	if o == nil {
		return OrderDetail{}
	}
	var completed string
	if t, ok := o.CompletedAt.Unwrap(); ok {
		completed = formatTime(t)
	}
	return OrderDetail{
		ID:          o.ID.String(),
		MenuID:      o.MenuID.String(),
		Status:      string(o.Status),
		CreatedAt:   formatTime(o.CreatedAt),
		CompletedAt: completed,
		Notes:       o.Notes,
	}
}

func ToOrderItemRows(items []models.OrderItem) []OrderItemRow {
	rows := make([]OrderItemRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, OrderItemRow{
			DrinkID:  item.DrinkID.String(),
			Quantity: item.Quantity,
		})
	}
	return rows
}

func TemplatePlace() OrderInput {
	return OrderInput{
		MenuID: "mnu-abc123",
		Items: []OrderItemRow{
			{DrinkID: "drk-abc123", Quantity: 2},
			{DrinkID: "drk-def456", Quantity: 1},
		},
	}
}

func DecodePlace(r io.Reader) (*models.Order, error) {
	var input OrderInput
	if err := json.NewDecoder(r).Decode(&input); err != nil {
		return nil, errors.Invalidf("parse order json: %w", err)
	}
	return input.ToDomain()
}

func (input OrderInput) ToDomain() (*models.Order, error) {
	if input.MenuID == "" {
		return nil, errors.Invalidf("menu_id is required")
	}
	menuID, err := entity.ParseMenuID(input.MenuID)
	if err != nil {
		return nil, errors.Invalidf("invalid menu id %q: %w", input.MenuID, err)
	}
	items := make([]models.OrderItem, 0, len(input.Items))
	for i, item := range input.Items {
		drinkID, err := entity.ParseDrinkID(item.DrinkID)
		if err != nil {
			return nil, errors.Invalidf("invalid item %d drink id %q: %w", i, item.DrinkID, err)
		}
		items = append(items, models.OrderItem{
			DrinkID:  drinkID,
			Quantity: item.Quantity,
		})
	}
	return &models.Order{
		MenuID: menuID,
		Items:  items,
	}, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
