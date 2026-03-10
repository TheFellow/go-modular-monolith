package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

//go:embed data/ingredients.json
var ingredientsJSON []byte

//go:embed data/drinks.json
var drinksJSON []byte

// JSON structures for parsing seed data

type seedIngredient struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Unit        string `json:"unit"`
	Description string `json:"description"`
	Stock       struct {
		Quantity float64 `json:"quantity"`
		Cost     string  `json:"cost"`
	} `json:"stock"`
}

type seedDrink struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Glass       string `json:"glass"`
	Description string `json:"description"`
	Recipe      struct {
		Ingredients []struct {
			Key    string  `json:"key"`
			Amount float64 `json:"amount"`
			Unit   string  `json:"unit"`
		} `json:"ingredients"`
		Steps   []string `json:"steps"`
		Garnish string   `json:"garnish"`
	} `json:"recipe"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("=== Mixology Seed ===")
	fmt.Println()

	// Open store
	dbPath := "data/mixology.db"
	if p := os.Getenv("MIXOLOGY_DB"); p != "" {
		dbPath = p
	}

	s, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer s.Close()

	// Create app
	a := app.New(
		app.WithStore(s),
		app.WithPrincipal(authn.Owner()),
	)
	defer a.Close()

	// Create context as owner
	ctx := a.Context()

	// Parse JSON data
	var ingredients []seedIngredient
	if err := json.Unmarshal(ingredientsJSON, &ingredients); err != nil {
		return fmt.Errorf("parse ingredients.json: %w", err)
	}

	var drinks []seedDrink
	if err := json.Unmarshal(drinksJSON, &drinks); err != nil {
		return fmt.Errorf("parse drinks.json: %w", err)
	}

	// Map from key -> ingredient ID for linking recipes
	ingredientIDs := make(map[string]entity.IngredientID)

	// Create ingredients
	fmt.Println("Creating ingredients...")
	for _, ing := range ingredients {
		ingredient := &ingredientmodels.Ingredient{
			Name:        ing.Name,
			Category:    ingredientmodels.Category(ing.Category),
			Unit:        measurement.Unit(ing.Unit),
			Description: ing.Description,
		}

		created, err := a.Ingredients.Create(ctx, ingredient)
		if err != nil {
			return fmt.Errorf("create ingredient %q: %w", ing.Name, err)
		}

		ingredientIDs[ing.Key] = created.ID
		fmt.Printf("  %s: %s\n", created.Name, created.ID)
	}
	fmt.Printf("  Created %d ingredients\n", len(ingredients))

	// Set inventory levels
	fmt.Println()
	fmt.Println("Setting inventory levels...")
	for _, ing := range ingredients {
		ingID := ingredientIDs[ing.Key]

		amount, err := measurement.NewAmount(ing.Stock.Quantity, measurement.Unit(ing.Unit))
		if err != nil {
			return fmt.Errorf("parse amount for %q: %w", ing.Name, err)
		}

		cost, err := parseCost(ing.Stock.Cost)
		if err != nil {
			return fmt.Errorf("parse cost for %q: %w", ing.Name, err)
		}

		update := &inventorymodels.Update{
			IngredientID: ingID,
			Amount:       amount,
			CostPerUnit:  cost,
		}

		if _, err := a.Inventory.Set(ctx, update); err != nil {
			return fmt.Errorf("set inventory for %q: %w", ing.Name, err)
		}
	}
	fmt.Println("  Inventory stocked")

	// Create drinks
	fmt.Println()
	fmt.Println("Creating drinks...")
	var drinkIDs []entity.DrinkID
	for _, d := range drinks {
		// Build recipe ingredients
		recipeIngredients := make([]drinksmodels.RecipeIngredient, 0, len(d.Recipe.Ingredients))
		for _, ri := range d.Recipe.Ingredients {
			ingID, ok := ingredientIDs[ri.Key]
			if !ok {
				return fmt.Errorf("unknown ingredient key %q in drink %q", ri.Key, d.Name)
			}

			amount, err := measurement.NewAmount(ri.Amount, measurement.Unit(ri.Unit))
			if err != nil {
				return fmt.Errorf("parse amount for ingredient %q in drink %q: %w", ri.Key, d.Name, err)
			}

			recipeIngredients = append(recipeIngredients, drinksmodels.RecipeIngredient{
				IngredientID: ingID,
				Amount:       amount,
			})
		}

		drink := &drinksmodels.Drink{
			Name:        d.Name,
			Category:    drinksmodels.DrinkCategory(d.Category),
			Glass:       drinksmodels.GlassType(d.Glass),
			Description: d.Description,
			Recipe: drinksmodels.Recipe{
				Ingredients: recipeIngredients,
				Steps:       d.Recipe.Steps,
				Garnish:     d.Recipe.Garnish,
			},
		}

		created, err := a.Drinks.Create(ctx, drink)
		if err != nil {
			return fmt.Errorf("create drink %q: %w", d.Name, err)
		}

		drinkIDs = append(drinkIDs, created.ID)
		fmt.Printf("  %s: %s\n", created.Name, created.ID)
	}

	// Create menu
	fmt.Println()
	fmt.Println("Creating menu...")
	menu := &menumodels.Menu{
		Name: "Classic Cocktails",
	}

	createdMenu, err := a.Menu.Create(ctx, menu)
	if err != nil {
		return fmt.Errorf("create menu: %w", err)
	}
	fmt.Printf("  Menu: %s\n", createdMenu.ID)

	// Add drinks to menu
	for _, drinkID := range drinkIDs {
		patch := &menumodels.MenuPatch{
			MenuID:  createdMenu.ID,
			DrinkID: drinkID,
		}
		if _, err := a.Menu.AddDrink(ctx, patch); err != nil {
			return fmt.Errorf("add drink to menu: %w", err)
		}
	}

	// Publish menu
	if _, err := a.Menu.Publish(ctx, createdMenu); err != nil {
		return fmt.Errorf("publish menu: %w", err)
	}
	fmt.Printf("  Menu published with %d drinks\n", len(drinkIDs))

	// Summary
	fmt.Println()
	fmt.Println("=== Seed Complete ===")
	fmt.Println()
	fmt.Println("Created:")
	fmt.Printf("  - %d ingredients\n", len(ingredients))
	fmt.Printf("  - %d classic cocktails\n", len(drinkIDs))
	fmt.Println("  - 1 published menu")
	fmt.Println()
	fmt.Println("View the menu with cost analysis:")
	fmt.Printf("  mixology menu show --id %s --costs --target-margin 0.7\n", createdMenu.ID)
	fmt.Println()
	fmt.Println("List all drinks:")
	fmt.Println("  mixology drinks list")
	fmt.Println()
	fmt.Println("Check inventory:")
	fmt.Println("  mixology inventory list")

	return nil
}

// parseCost parses a cost string like "$28.00" into a Price
func parseCost(s string) (money.Price, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return money.Price{}, fmt.Errorf("empty cost")
	}

	// Remove currency symbol
	s = strings.TrimPrefix(s, "$")
	s = strings.TrimPrefix(s, "€")
	s = strings.TrimSpace(s)

	return money.NewPrice(s, currency.USD)
}
