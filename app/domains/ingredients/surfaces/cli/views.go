package cli

import (
	"encoding/json"
	"io"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type IngredientRow struct {
	ID       string `table:"ID" json:"id,omitempty"`
	Name     string `table:"NAME" json:"name"`
	Category string `table:"CATEGORY" json:"category"`
	Unit     string `table:"UNIT" json:"unit"`
	Desc     string `table:"-" json:"description,omitempty"`
}

func ToIngredientRow(i *models.Ingredient) IngredientRow {
	if i == nil {
		return IngredientRow{}
	}
	return IngredientRow{
		ID:       i.ID.String(),
		Name:     i.Name,
		Category: string(i.Category),
		Unit:     string(i.Unit),
		Desc:     i.Description,
	}
}

func ToIngredientRows(items []*models.Ingredient) []IngredientRow {
	rows := make([]IngredientRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, ToIngredientRow(item))
	}
	return rows
}

func TemplateCreate() IngredientRow {
	return IngredientRow{
		Name:     "Vodka",
		Category: string(models.CategorySpirit),
		Unit:     string(measurement.UnitOz),
		Desc:     "Neutral grain spirit",
	}
}

func TemplateUpdate() IngredientRow {
	return IngredientRow{
		ID:       "ing-abc123",
		Name:     "Vodka",
		Category: string(models.CategorySpirit),
		Unit:     string(measurement.UnitOz),
		Desc:     "Neutral grain spirit",
	}
}

func DecodeCreate(r io.Reader) (*models.Ingredient, error) {
	var row IngredientRow
	if err := json.NewDecoder(r).Decode(&row); err != nil {
		return nil, errors.Invalidf("parse ingredient json: %w", err)
	}
	return &models.Ingredient{
		Name:        row.Name,
		Category:    models.Category(row.Category),
		Unit:        measurement.Unit(row.Unit),
		Description: row.Desc,
	}, nil
}

func DecodeUpdate(r io.Reader) (*models.Ingredient, error) {
	var row IngredientRow
	if err := json.NewDecoder(r).Decode(&row); err != nil {
		return nil, errors.Invalidf("parse ingredient json: %w", err)
	}
	if row.ID == "" {
		return nil, errors.Invalidf("id is required")
	}
	parsedID, err := entity.ParseIngredientID(row.ID)
	if err != nil {
		return nil, errors.Invalidf("invalid ingredient id %q: %w", row.ID, err)
	}
	return &models.Ingredient{
		ID:          parsedID,
		Name:        row.Name,
		Category:    models.Category(row.Category),
		Unit:        measurement.Unit(row.Unit),
		Description: row.Desc,
	}, nil
}
