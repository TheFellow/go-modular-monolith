package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type InventoryRow struct {
	IngredientID string   `table:"INGREDIENT_ID" json:"ingredient_id"`
	Quantity     Quantity `table:"QUANTITY" json:"quantity"`
	Unit         string   `table:"UNIT" json:"unit"`
	CostPerUnit  string   `table:"-" json:"cost_per_unit,omitempty"`
}

type InventoryInput struct {
	IngredientID string   `json:"ingredient_id"`
	Quantity     *float64 `json:"quantity"`
	Unit         string   `json:"unit,omitempty"`
	CostPerUnit  string   `json:"cost_per_unit,omitempty"`
}

type InventoryPatch struct {
	IngredientID string   `json:"ingredient_id"`
	Delta        *float64 `json:"delta,omitempty"`
	Reason       string   `json:"reason"`
	CostPerUnit  string   `json:"cost_per_unit,omitempty"`
}

func ToInventoryRow(s *models.Inventory) InventoryRow {
	if s == nil {
		return InventoryRow{}
	}
	return InventoryRow{
		IngredientID: s.IngredientID.String(),
		Quantity:     Quantity(s.Amount.Value()),
		Unit:         string(s.Amount.Unit()),
	}
}

func ToInventoryRows(items []*models.Inventory) []InventoryRow {
	rows := make([]InventoryRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, ToInventoryRow(item))
	}
	return rows
}

func TemplateSet() InventoryInput {
	quantity := 25.0
	return InventoryInput{
		IngredientID: "ing-abc123",
		Quantity:     &quantity,
		Unit:         string(measurement.UnitOz),
		CostPerUnit:  "$28.00",
	}
}

func DecodeSet(r io.Reader) (*InventoryInput, error) {
	var input InventoryInput
	if err := json.NewDecoder(r).Decode(&input); err != nil {
		return nil, errors.Invalidf("parse inventory json: %w", err)
	}
	if input.IngredientID == "" {
		return nil, errors.Invalidf("ingredient_id is required")
	}
	if _, err := entity.ParseIngredientID(input.IngredientID); err != nil {
		return nil, errors.Invalidf("invalid ingredient id %q: %w", input.IngredientID, err)
	}
	if input.Quantity == nil {
		return nil, errors.Invalidf("quantity is required")
	}
	if unit := strings.TrimSpace(input.Unit); unit != "" {
		if err := measurement.Unit(unit).Validate(); err != nil {
			return nil, err
		}
	}
	if input.CostPerUnit != "" {
		if _, err := parsePrice(input.CostPerUnit); err != nil {
			return nil, err
		}
	}
	return &input, nil
}

func TemplateAdjust() InventoryPatch {
	delta := 5.0
	return InventoryPatch{
		IngredientID: "ing-abc123",
		Delta:        &delta,
		Reason:       "received",
		CostPerUnit:  "$28.00",
	}
}

func DecodeAdjust(r io.Reader) (*InventoryPatch, error) {
	var input InventoryPatch
	if err := json.NewDecoder(r).Decode(&input); err != nil {
		return nil, errors.Invalidf("parse inventory patch json: %w", err)
	}
	if input.IngredientID == "" {
		return nil, errors.Invalidf("ingredient_id is required")
	}
	if _, err := entity.ParseIngredientID(input.IngredientID); err != nil {
		return nil, errors.Invalidf("invalid ingredient id %q: %w", input.IngredientID, err)
	}
	if strings.TrimSpace(input.Reason) == "" {
		return nil, errors.Invalidf("reason is required")
	}
	if input.Delta == nil && input.CostPerUnit == "" {
		return nil, errors.Invalidf("at least one of delta or cost_per_unit is required")
	}
	if input.CostPerUnit != "" {
		if _, err := parsePrice(input.CostPerUnit); err != nil {
			return nil, err
		}
	}
	return &input, nil
}

type Quantity float64

func (q Quantity) String() string {
	return fmt.Sprintf("%.2f", q)
}

func parsePrice(s string) (money.Price, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return money.Price{}, errors.Invalidf("price is required")
	}
	if strings.HasPrefix(s, "$") {
		return money.NewPrice(strings.TrimPrefix(s, "$"), currency.USD)
	}
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return money.Price{}, errors.Invalidf("invalid price %q (expected \"$1.23\" or \"USD 1.23\" or \"1.23 USD\")", s)
	}

	var currencyCode, number string
	if isCurrency(parts[0]) {
		currencyCode, number = parts[0], parts[1]
	} else if isCurrency(parts[1]) {
		currencyCode, number = parts[1], parts[0]
	} else {
		return money.Price{}, errors.Invalidf("invalid price %q (expected \"$1.23\" or \"USD 1.23\" or \"1.23 USD\")", s)
	}

	curr, err := currency.Parse(currencyCode)
	if err != nil {
		return money.Price{}, err
	}
	return money.NewPrice(number, curr)
}

func isCurrency(s string) bool {
	if len(s) != 3 {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			if r < 'a' || r > 'z' {
				return false
			}
		}
	}
	return true
}
