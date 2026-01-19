package main

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	ingredientscli "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/cli"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
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
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "category",
						Aliases: []string{"c"},
						Usage:   ingredientscli.CategoryUsage(),
						Validator: func(s string) error {
							return ingredientscli.ValidateCategory(s)
						},
					},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					res, err := c.app.Ingredients.List(ctx, ingredients.ListRequest{
						Category: models.Category(cmd.String("category")),
					})
					if err != nil {
						return err
					}

					for _, i := range res {
						fmt.Printf("%s\t%s\t%s\t%s\n", i.ID.String(), i.Name, i.Category, i.Unit)
					}
					return nil
				}),
			},
			{
				Name:  "get",
				Usage: "Get an ingredient by ID",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Ingredient ID", Required: true},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					ingredientID, err := entity.ParseIngredientID(cmd.String("id"))
					if err != nil {
						return err
					}
					res, err := c.app.Ingredients.Get(ctx, ingredientID)
					if err != nil {
						return err
					}

					i := res
					fmt.Printf("ID:          %s\n", i.ID.String())
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
						Usage:    ingredientscli.CategoryUsage(),
						Required: true,
						Validator: func(s string) error {
							return ingredientscli.ValidateCategory(s)
						},
					},
					&cli.StringFlag{
						Name:     "unit",
						Aliases:  []string{"u"},
						Usage:    ingredientscli.UnitUsage(),
						Required: true,
						Validator: func(s string) error {
							return ingredientscli.ValidateUnit(s)
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
					res, err := c.app.Ingredients.Create(ctx, &models.Ingredient{
						Name:        name,
						Category:    models.Category(cmd.String("category")),
						Unit:        models.Unit(cmd.String("unit")),
						Description: cmd.String("description"),
					})
					if err != nil {
						return err
					}

					fmt.Printf("%s\t%s\t%s\t%s\n", res.ID.String(), res.Name, res.Category, res.Unit)
					return nil
				}),
			},
			{
				Name:  "update",
				Usage: "Update an ingredient",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Ingredient ID", Required: true},
					&cli.StringFlag{
						Name:    "name",
						Aliases: []string{"n"},
						Usage:   "New name",
					},
					&cli.StringFlag{
						Name:    "category",
						Aliases: []string{"c"},
						Usage:   ingredientscli.CategoryUsage(),
						Validator: func(s string) error {
							return ingredientscli.ValidateCategory(s)
						},
					},
					&cli.StringFlag{
						Name:    "unit",
						Aliases: []string{"u"},
						Usage:   ingredientscli.UnitUsage(),
						Validator: func(s string) error {
							return ingredientscli.ValidateUnit(s)
						},
					},
					&cli.StringFlag{
						Name:    "description",
						Aliases: []string{"d"},
						Usage:   "Description",
					},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					ingredientID, err := entity.ParseIngredientID(cmd.String("id"))
					if err != nil {
						return err
					}
					res, err := c.app.Ingredients.Update(ctx, &models.Ingredient{
						ID:          ingredientID,
						Name:        cmd.String("name"),
						Category:    models.Category(cmd.String("category")),
						Unit:        models.Unit(cmd.String("unit")),
						Description: cmd.String("description"),
					})
					if err != nil {
						return err
					}

					fmt.Printf("%s\t%s\t%s\t%s\n", res.ID.String(), res.Name, res.Category, res.Unit)
					return nil
				}),
			},
			{
				Name:  "delete",
				Usage: "Delete an ingredient by ID",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Ingredient ID", Required: true},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					ingredientID, err := entity.ParseIngredientID(cmd.String("id"))
					if err != nil {
						return err
					}
					res, err := c.app.Ingredients.Delete(ctx, ingredientID)
					if err != nil {
						return err
					}

					fmt.Printf("deleted %s\t%s\n", res.ID.String(), res.Name)
					return nil
				}),
			},
		},
	}
}
