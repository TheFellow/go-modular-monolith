package main

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	ingredientscli "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/cli"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	clitable "github.com/TheFellow/go-modular-monolith/main/cli/table"
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
					JSONFlag,
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

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, res)
					}

					return clitable.PrintTable(ingredientscli.ToIngredientRows(res))
				}),
			},
			{
				Name:  "get",
				Usage: "Get an ingredient by ID",
				Flags: []cli.Flag{
					JSONFlag,
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

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, res)
					}

					return clitable.PrintDetail(ingredientscli.ToIngredientRow(res))
				}),
			},
			{
				Name:  "create",
				Usage: "Create a new ingredient",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "name", UsageText: "Ingredient name", Min: 1, Max: 1},
				},
				Flags: []cli.Flag{
					JSONFlag,
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
						Unit:        measurement.Unit(cmd.String("unit")),
						Description: cmd.String("description"),
					})
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, res)
					}

					fmt.Println(res.ID.String())
					return nil
				}),
			},
			{
				Name:  "update",
				Usage: "Update an ingredient",
				Flags: []cli.Flag{
					JSONFlag,
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
						Unit:        measurement.Unit(cmd.String("unit")),
						Description: cmd.String("description"),
					})
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, res)
					}

					fmt.Println(res.ID.String())
					return nil
				}),
			},
			{
				Name:  "delete",
				Usage: "Delete an ingredient by ID",
				Flags: []cli.Flag{
					JSONFlag,
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

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, res)
					}

					fmt.Println(res.ID.String())
					return nil
				}),
			},
		},
	}
}
