package cli

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type Drink struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category,omitempty"`
	Glass       string `json:"glass,omitempty"`
	Description string `json:"description,omitempty"`
	Recipe      Recipe `json:"recipe"`
}

func FromDomainDrink(d models.Drink) Drink {
	return Drink{
		ID:          string(d.ID.ID),
		Name:        d.Name,
		Category:    string(d.Category),
		Glass:       string(d.Glass),
		Description: d.Description,
		Recipe:      FromDomainRecipe(d.Recipe),
	}
}

func TemplateUpdateDrink() Drink {
	return Drink{
		ID:          "margarita",
		Name:        "Margarita",
		Category:    string(models.DrinkCategoryCocktail),
		Glass:       string(models.GlassTypeCoupe),
		Description: "A classic sour",
		Recipe:      TemplateRecipe(),
	}
}

func DecodeUpdateDrinkJSON(r io.Reader) (models.Drink, error) {
	var doc Drink
	if err := json.NewDecoder(r).Decode(&doc); err != nil {
		return models.Drink{}, errors.Invalidf("parse drink json: %w", err)
	}
	return doc.ToDomainForUpdate()
}

func (d Drink) ToDomainForUpdate() (models.Drink, error) {
	id := strings.TrimSpace(d.ID)
	if id == "" {
		return models.Drink{}, errors.Invalidf("id is required")
	}

	recipe, err := d.Recipe.ToDomain()
	if err != nil {
		return models.Drink{}, err
	}

	out := models.Drink{
		ID:          models.NewDrinkID(id),
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
