package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/drinks"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/inventory"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/urfave/cli/v3"
)

type recipeJSON struct {
	Ingredients []recipeIngredientJSON `json:"ingredients"`
	Steps       []string               `json:"steps"`
	Garnish     string                 `json:"garnish,omitempty"`
}

type recipeIngredientJSON struct {
	IngredientID string   `json:"ingredient_id"`
	Amount       float64  `json:"amount"`
	Unit         string   `json:"unit"`
	Optional     bool     `json:"optional,omitempty"`
	Substitutes  []string `json:"substitutes,omitempty"`
}

func parseRecipeFromFlags(cmd *cli.Command) (drinksmodels.Recipe, error) {
	recipeStr := strings.TrimSpace(cmd.String("recipe"))
	recipeFile := strings.TrimSpace(cmd.String("recipe-file"))

	if recipeStr == "" && recipeFile == "" {
		return drinksmodels.Recipe{}, fmt.Errorf("missing recipe: set --recipe or --recipe-file")
	}
	if recipeStr != "" && recipeFile != "" {
		return drinksmodels.Recipe{}, fmt.Errorf("set only one of --recipe or --recipe-file")
	}

	if recipeFile != "" {
		b, err := os.ReadFile(recipeFile)
		if err != nil {
			return drinksmodels.Recipe{}, err
		}
		recipeStr = string(b)
	}

	var rj recipeJSON
	if err := json.Unmarshal([]byte(recipeStr), &rj); err != nil {
		return drinksmodels.Recipe{}, fmt.Errorf("parse recipe json: %w", err)
	}

	ingredients := make([]drinksmodels.RecipeIngredient, 0, len(rj.Ingredients))
	for _, ing := range rj.Ingredients {
		subs := make([]cedar.EntityUID, 0, len(ing.Substitutes))
		for _, sub := range ing.Substitutes {
			subs = append(subs, models.NewIngredientID(sub))
		}

		ingredients = append(ingredients, drinksmodels.RecipeIngredient{
			IngredientID: models.NewIngredientID(ing.IngredientID),
			Amount:       ing.Amount,
			Unit:         models.Unit(ing.Unit),
			Optional:     ing.Optional,
			Substitutes:  subs,
		})
	}

	return drinksmodels.Recipe{
		Ingredients: ingredients,
		Steps:       rj.Steps,
		Garnish:     rj.Garnish,
	}, nil
}

func buildApp() *cli.Command {
	var a *app.App
	var actor string

	cmd := &cli.Command{
		Name:  "mixology",
		Usage: "Mixology as a Service",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "as",
				Usage:       "Actor to run as (owner|anonymous)",
				Value:       "owner",
				Destination: &actor,
			},
		},
		Before: func(ctx context.Context, _ *cli.Command) (context.Context, error) {
			var err error
			if a == nil {
				a, err = app.New()
				if err != nil {
					return ctx, err
				}
			}

			p, err := authn.ParseActor(actor)
			if err != nil {
				return ctx, err
			}
			return middleware.NewContext(ctx, middleware.WithPrincipal(p)), nil
		},
		Commands: []*cli.Command{
			{
				Name:  "drinks",
				Usage: "Manage drinks",
				Commands: []*cli.Command{
					{
						Name:  "list",
						Usage: "List drinks",
						Action: func(ctx context.Context, _ *cli.Command) error {
							mctx, ok := ctx.(*middleware.Context)
							if !ok {
								return fmt.Errorf("expected middleware context")
							}

							res, err := a.Drinks().List(mctx, drinks.ListRequest{})
							if err != nil {
								return err
							}

							for _, d := range res.Drinks {
								fmt.Printf("%s\t%s\n", string(d.ID.ID), d.Name)
							}
							return nil
						},
					},
					{
						Name:      "get",
						Usage:     "Get a drink by ID",
						ArgsUsage: "<id>",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							id := cmd.Args().First()
							if id == "" {
								return fmt.Errorf("missing id")
							}

							mctx, ok := ctx.(*middleware.Context)
							if !ok {
								return fmt.Errorf("expected middleware context")
							}

							res, err := a.Drinks().Get(mctx, drinks.GetRequest{ID: drinksmodels.NewDrinkID(id)})
							if err != nil {
								return err
							}

							fmt.Printf("%s\t%s\n", string(res.Drink.ID.ID), res.Drink.Name)
							return nil
						},
					},
					{
						Name:      "create",
						Usage:     "Create a new drink",
						ArgsUsage: "<name>",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "category",
								Aliases: []string{"c"},
								Usage:   "Category (cocktail|mocktail|shot|highball|martini|sour|tiki)",
							},
							&cli.StringFlag{
								Name:    "glass",
								Aliases: []string{"g"},
								Usage:   "Glass (rocks|highball|coupe|martini)",
							},
							&cli.StringFlag{
								Name:    "description",
								Aliases: []string{"d"},
								Usage:   "Description",
							},
							&cli.StringFlag{
								Name:  "recipe",
								Usage: "Recipe JSON string (see --recipe-file to avoid quoting)",
							},
							&cli.StringFlag{
								Name:  "recipe-file",
								Usage: "Path to JSON file containing the recipe",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							name := strings.TrimSpace(strings.Join(cmd.Args().Slice(), " "))
							if name == "" {
								return fmt.Errorf("missing name")
							}

							mctx, ok := ctx.(*middleware.Context)
							if !ok {
								return fmt.Errorf("expected middleware context")
							}

							recipe, err := parseRecipeFromFlags(cmd)
							if err != nil {
								return err
							}

							res, err := a.Drinks().Create(mctx, drinks.CreateRequest{
								Name:        name,
								Category:    drinksmodels.DrinkCategory(cmd.String("category")),
								Glass:       drinksmodels.GlassType(cmd.String("glass")),
								Recipe:      recipe,
								Description: cmd.String("description"),
							})
							if err != nil {
								return err
							}

							fmt.Printf("%s\t%s\n", string(res.Drink.ID.ID), res.Drink.Name)
							return nil
						},
					},
					{
						Name:      "update-recipe",
						Usage:     "Update a drink's recipe",
						ArgsUsage: "<id>",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "recipe",
								Usage: "Recipe JSON string (see --recipe-file to avoid quoting)",
							},
							&cli.StringFlag{
								Name:  "recipe-file",
								Usage: "Path to JSON file containing the recipe",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							id := cmd.Args().First()
							if id == "" {
								return fmt.Errorf("missing id")
							}

							mctx, ok := ctx.(*middleware.Context)
							if !ok {
								return fmt.Errorf("expected middleware context")
							}

							recipe, err := parseRecipeFromFlags(cmd)
							if err != nil {
								return err
							}

							res, err := a.Drinks().UpdateRecipe(mctx, drinks.UpdateRecipeRequest{
								ID:     drinksmodels.NewDrinkID(id),
								Recipe: recipe,
							})
							if err != nil {
								return err
							}

							fmt.Printf("%s\t%s\n", string(res.Drink.ID.ID), res.Drink.Name)
							return nil
						},
					},
				},
			},
			{
				Name:  "ingredients",
				Usage: "Manage ingredients",
				Commands: []*cli.Command{
					{
						Name:  "list",
						Usage: "List ingredients",
						Action: func(ctx context.Context, _ *cli.Command) error {
							mctx, ok := ctx.(*middleware.Context)
							if !ok {
								return fmt.Errorf("expected middleware context")
							}

							res, err := a.Ingredients().List(mctx, ingredients.ListRequest{})
							if err != nil {
								return err
							}

							for _, i := range res.Ingredients {
								fmt.Printf("%s\t%s\t%s\t%s\n", string(i.ID.ID), i.Name, i.Category, i.Unit)
							}
							return nil
						},
					},
					{
						Name:      "get",
						Usage:     "Get an ingredient by ID",
						ArgsUsage: "<id>",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							id := cmd.Args().First()
							if id == "" {
								return fmt.Errorf("missing id")
							}

							mctx, ok := ctx.(*middleware.Context)
							if !ok {
								return fmt.Errorf("expected middleware context")
							}

							res, err := a.Ingredients().Get(mctx, ingredients.GetRequest{ID: models.NewIngredientID(id)})
							if err != nil {
								return err
							}

							i := res.Ingredient
							fmt.Printf("ID:          %s\n", string(i.ID.ID))
							fmt.Printf("Name:        %s\n", i.Name)
							fmt.Printf("Category:    %s\n", i.Category)
							fmt.Printf("Unit:        %s\n", i.Unit)
							if i.Description != "" {
								fmt.Printf("Description: %s\n", i.Description)
							}
							return nil
						},
					},
					{
						Name:      "create",
						Usage:     "Create a new ingredient",
						ArgsUsage: "<name>",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "category",
								Aliases:  []string{"c"},
								Usage:    "Category (spirit|mixer|garnish|bitter|syrup|juice|other)",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "unit",
								Aliases:  []string{"u"},
								Usage:    "Unit (oz|ml|dash|piece|splash)",
								Required: true,
							},
							&cli.StringFlag{
								Name:    "description",
								Aliases: []string{"d"},
								Usage:   "Description",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							name := strings.TrimSpace(strings.Join(cmd.Args().Slice(), " "))
							if name == "" {
								return fmt.Errorf("missing name")
							}

							mctx, ok := ctx.(*middleware.Context)
							if !ok {
								return fmt.Errorf("expected middleware context")
							}

							res, err := a.Ingredients().Create(mctx, ingredients.CreateRequest{
								Name:        name,
								Category:    models.Category(cmd.String("category")),
								Unit:        models.Unit(cmd.String("unit")),
								Description: cmd.String("description"),
							})
							if err != nil {
								return err
							}

							i := res.Ingredient
							fmt.Printf("%s\t%s\t%s\t%s\n", string(i.ID.ID), i.Name, i.Category, i.Unit)
							return nil
						},
					},
					{
						Name:      "update",
						Usage:     "Update an ingredient",
						ArgsUsage: "<id>",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "name",
								Aliases: []string{"n"},
								Usage:   "New name",
							},
							&cli.StringFlag{
								Name:    "category",
								Aliases: []string{"c"},
								Usage:   "Category (spirit|mixer|garnish|bitter|syrup|juice|other)",
							},
							&cli.StringFlag{
								Name:    "unit",
								Aliases: []string{"u"},
								Usage:   "Unit (oz|ml|dash|piece|splash)",
							},
							&cli.StringFlag{
								Name:    "description",
								Aliases: []string{"d"},
								Usage:   "Description",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							id := cmd.Args().First()
							if id == "" {
								return fmt.Errorf("missing id")
							}

							mctx, ok := ctx.(*middleware.Context)
							if !ok {
								return fmt.Errorf("expected middleware context")
							}

							res, err := a.Ingredients().Update(mctx, ingredients.UpdateRequest{
								ID:          models.NewIngredientID(id),
								Name:        cmd.String("name"),
								Category:    models.Category(cmd.String("category")),
								Unit:        models.Unit(cmd.String("unit")),
								Description: cmd.String("description"),
							})
							if err != nil {
								return err
							}

							i := res.Ingredient
							fmt.Printf("%s\t%s\t%s\t%s\n", string(i.ID.ID), i.Name, i.Category, i.Unit)
							return nil
						},
					},
				},
			},
			{
				Name:  "inventory",
				Usage: "Manage ingredient stock",
				Commands: []*cli.Command{
					{
						Name:  "list",
						Usage: "List stock levels",
						Action: func(ctx context.Context, _ *cli.Command) error {
							mctx, ok := ctx.(*middleware.Context)
							if !ok {
								return fmt.Errorf("expected middleware context")
							}

							res, err := a.Inventory().List(mctx, inventory.ListRequest{})
							if err != nil {
								return err
							}

							for _, s := range res.Stock {
								fmt.Printf("%s\t%.2f\t%s\n", string(s.IngredientID.ID), s.Quantity, s.Unit)
							}
							return nil
						},
					},
					{
						Name:      "get",
						Usage:     "Get stock for an ingredient",
						ArgsUsage: "<ingredient-id>",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							id := cmd.Args().First()
							if id == "" {
								return fmt.Errorf("missing ingredient-id")
							}

							mctx, ok := ctx.(*middleware.Context)
							if !ok {
								return fmt.Errorf("expected middleware context")
							}

							res, err := a.Inventory().Get(mctx, inventory.GetRequest{IngredientID: models.NewIngredientID(id)})
							if err != nil {
								return err
							}

							s := res.Stock
							fmt.Printf("%s\t%.2f\t%s\n", string(s.IngredientID.ID), s.Quantity, s.Unit)
							return nil
						},
					},
					{
						Name:      "adjust",
						Usage:     "Adjust stock by delta",
						ArgsUsage: "<ingredient-id> <delta>",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "reason",
								Aliases:  []string{"r"},
								Usage:    "Reason (received|used|spilled|expired|corrected)",
								Required: true,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							ingredientID := cmd.Args().Get(0)
							deltaStr := cmd.Args().Get(1)
							if ingredientID == "" || deltaStr == "" {
								return fmt.Errorf("usage: inventory adjust <ingredient-id> <delta> --reason=<reason>")
							}

							delta, err := strconv.ParseFloat(deltaStr, 64)
							if err != nil {
								return fmt.Errorf("invalid delta: %w", err)
							}

							mctx, ok := ctx.(*middleware.Context)
							if !ok {
								return fmt.Errorf("expected middleware context")
							}

							res, err := a.Inventory().Adjust(mctx, inventory.AdjustRequest{
								IngredientID: models.NewIngredientID(ingredientID),
								Delta:        delta,
								Reason:       inventorymodels.AdjustmentReason(cmd.String("reason")),
							})
							if err != nil {
								return err
							}

							s := res.Stock
							fmt.Printf("%s\t%.2f\t%s\n", string(s.IngredientID.ID), s.Quantity, s.Unit)
							return nil
						},
					},
					{
						Name:      "set",
						Usage:     "Set stock quantity",
						ArgsUsage: "<ingredient-id> <quantity>",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							ingredientID := cmd.Args().Get(0)
							qtyStr := cmd.Args().Get(1)
							if ingredientID == "" || qtyStr == "" {
								return fmt.Errorf("usage: inventory set <ingredient-id> <quantity>")
							}

							qty, err := strconv.ParseFloat(qtyStr, 64)
							if err != nil {
								return fmt.Errorf("invalid quantity: %w", err)
							}

							mctx, ok := ctx.(*middleware.Context)
							if !ok {
								return fmt.Errorf("expected middleware context")
							}

							res, err := a.Inventory().Set(mctx, inventory.SetRequest{
								IngredientID: models.NewIngredientID(ingredientID),
								Quantity:     qty,
							})
							if err != nil {
								return err
							}

							s := res.Stock
							fmt.Printf("%s\t%.2f\t%s\n", string(s.IngredientID.ID), s.Quantity, s.Unit)
							return nil
						},
					},
				},
			},
		},
	}

	return cmd
}

func main() {
	cmd := buildApp()
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		_, _ = fmt.Fprintln(cmd.ErrWriter, err)
		os.Exit(1)
	}
}
