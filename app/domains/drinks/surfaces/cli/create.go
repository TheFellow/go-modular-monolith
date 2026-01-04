package cli

import (
	"encoding/json"
	"io"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type CreateDrink struct {
	Name        string `json:"name"`
	Category    string `json:"category,omitempty"`
	Glass       string `json:"glass,omitempty"`
	Description string `json:"description,omitempty"`
	Recipe      Recipe `json:"recipe"`
}

func TemplateCreateDrink() CreateDrink {
	return CreateDrink{
		Name:        "Margarita",
		Category:    string(models.DrinkCategoryCocktail),
		Glass:       string(models.GlassTypeCoupe),
		Description: "A classic sour",
		Recipe:      TemplateRecipe(),
	}
}

func DecodeCreateDrinkJSON(r io.Reader) (models.Drink, error) {
	var doc CreateDrink
	if err := json.NewDecoder(r).Decode(&doc); err != nil {
		return models.Drink{}, errors.Invalidf("parse drink json: %w", err)
	}
	return doc.ToDomain()
}

func (d CreateDrink) ToDomain() (models.Drink, error) {
	recipe, err := d.Recipe.ToDomain()
	if err != nil {
		return models.Drink{}, err
	}

	out := models.Drink{
		Name:        d.Name,
		Category:    models.DrinkCategory(d.Category),
		Glass:       models.GlassType(d.Glass),
		Recipe:      recipe,
		Description: d.Description,
	}

	if err := out.Category.Validate(); err != nil {
		return models.Drink{}, err
	}
	if err := out.Glass.Validate(); err != nil {
		return models.Drink{}, err
	}

	return out, nil
}
