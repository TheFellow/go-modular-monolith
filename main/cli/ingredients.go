package main

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/urfave/cli/v3"
)

func (c *CLI) ingredientsCommands() *cli.Command {
	return &cli.Command{
		Name:  "ingredients",
		Usage: "Manage ingredients",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List ingredients",
				Action: c.action(func(ctx *middleware.Context, _ *cli.Command) error {
					res, err := c.app.Ingredients().List(ctx, ingredients.ListRequest{})
					if err != nil {
						return err
					}

					for _, i := range res.Ingredients {
						fmt.Printf("%s\t%s\t%s\t%s\n", string(i.ID.ID), i.Name, i.Category, i.Unit)
					}
					return nil
				}),
			},
			{
				Name:  "get",
				Usage: "Get an ingredient by ID",
				Arguments: []cli.Argument{
					&cli.StringArg{Name: "id", UsageText: "Ingredient ID"},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					id := cmd.StringArgs("id")[0]
					res, err := c.app.Ingredients().Get(ctx, ingredients.GetRequest{ID: models.NewIngredientID(id)})
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
				}),
			},
			{
				Name:  "create",
				Usage: "Create a new ingredient",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "name", UsageText: "Ingredient name", Min: 1, Max: 1},
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "category",
						Aliases:  []string{"c"},
						Usage:    "Category (spirit|mixer|garnish|bitter|syrup|juice|other)",
						Required: true,
						Validator: func(s string) error {
							return validateIngredientCategory(s, true)
						},
					},
					&cli.StringFlag{
						Name:     "unit",
						Aliases:  []string{"u"},
						Usage:    "Unit (oz|ml|dash|piece|splash)",
						Required: true,
						Validator: func(s string) error {
							return validateIngredientUnit(s, true)
						},
					},
					&cli.StringFlag{
						Name:    "description",
						Aliases: []string{"d"},
						Usage:   "Description",
					},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					name := cmd.StringArgs("name")[0]
					res, err := c.app.Ingredients().Create(ctx, ingredients.CreateRequest{
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
				}),
			},
			{
				Name:  "update",
				Usage: "Update an ingredient",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "id", UsageText: "Ingredient ID", Min: 1, Max: 1},
				},
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
						Validator: func(s string) error {
							return validateIngredientCategory(s, false)
						},
					},
					&cli.StringFlag{
						Name:    "unit",
						Aliases: []string{"u"},
						Usage:   "Unit (oz|ml|dash|piece|splash)",
						Validator: func(s string) error {
							return validateIngredientUnit(s, false)
						},
					},
					&cli.StringFlag{
						Name:    "description",
						Aliases: []string{"d"},
						Usage:   "Description",
					},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					res, err := c.app.Ingredients().Update(ctx, ingredients.UpdateRequest{
						ID:          models.NewIngredientID(cmd.StringArgs("id")[0]),
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
				}),
			},
		},
	}
}

func validateIngredientCategory(s string, required bool) error {
	if s == "" && !required {
		return nil
	}

	switch models.Category(s) {
	case models.CategorySpirit, models.CategoryMixer, models.CategoryGarnish, models.CategoryBitter, models.CategorySyrup, models.CategoryJuice, models.CategoryOther:
		return nil
	default:
		return fmt.Errorf("invalid category: %s", s)
	}
}

func validateIngredientUnit(s string, required bool) error {
	if s == "" && !required {
		return nil
	}

	switch models.Unit(s) {
	case models.UnitOz, models.UnitMl, models.UnitDash, models.UnitPiece, models.UnitSplash:
		return nil
	default:
		return fmt.Errorf("invalid unit: %s", s)
	}
}
