package main

import (
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	ingredientscli "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/cli"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	clitable "github.com/TheFellow/go-modular-monolith/main/cli/table"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
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
					&cli.StringArgs{Name: "name", UsageText: "Ingredient name", Max: 1},
				},
				Flags: []cli.Flag{
					JSONFlag,
					TemplateFlag,
					StdinFlag,
					FileFlag,
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
					if cmd.Bool("template") {
						return writeJSON(cmd.Writer, ingredientscli.TemplateCreate())
					}

					var input *models.Ingredient
					if cmd.Bool("stdin") || strings.TrimSpace(cmd.String("file")) != "" {
						row, err := readJSONInput[ingredientscli.IngredientRow](cmd)
						if err != nil {
							return err
						}
						input = &models.Ingredient{
							Name:        row.Name,
							Category:    models.Category(row.Category),
							Unit:        measurement.Unit(row.Unit),
							Description: row.Desc,
						}
					} else {
						args := cmd.StringArgs("name")
						if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
							return errors.Invalidf("name is required (or use --stdin/--file)")
						}
						category := strings.TrimSpace(cmd.String("category"))
						if category == "" {
							return errors.Invalidf("category is required (or use --stdin/--file)")
						}
						unit := strings.TrimSpace(cmd.String("unit"))
						if unit == "" {
							return errors.Invalidf("unit is required (or use --stdin/--file)")
						}
						input = &models.Ingredient{
							Name:        args[0],
							Category:    models.Category(category),
							Unit:        measurement.Unit(unit),
							Description: cmd.String("description"),
						}
					}

					res, err := c.app.Ingredients.Create(ctx, input)
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
					TemplateFlag,
					StdinFlag,
					FileFlag,
					&cli.StringFlag{Name: "id", Usage: "Ingredient ID"},
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
					if cmd.Bool("template") {
						return writeJSON(cmd.Writer, ingredientscli.TemplateUpdate())
					}

					var input *models.Ingredient
					if cmd.Bool("stdin") || strings.TrimSpace(cmd.String("file")) != "" {
						row, err := readJSONInput[ingredientscli.IngredientRow](cmd)
						if err != nil {
							return err
						}
						if strings.TrimSpace(row.ID) == "" {
							return errors.Invalidf("id is required")
						}
						ingredientID, err := entity.ParseIngredientID(row.ID)
						if err != nil {
							return err
						}
						input = &models.Ingredient{
							ID:          ingredientID,
							Name:        row.Name,
							Category:    models.Category(row.Category),
							Unit:        measurement.Unit(row.Unit),
							Description: row.Desc,
						}
					} else {
						id := strings.TrimSpace(cmd.String("id"))
						if id == "" {
							return errors.Invalidf("id is required (or use --stdin/--file)")
						}
						ingredientID, err := entity.ParseIngredientID(id)
						if err != nil {
							return err
						}
						input = &models.Ingredient{
							ID:          ingredientID,
							Name:        cmd.String("name"),
							Category:    models.Category(cmd.String("category")),
							Unit:        measurement.Unit(cmd.String("unit")),
							Description: cmd.String("description"),
						}
					}

					res, err := c.app.Ingredients.Update(ctx, input)
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
