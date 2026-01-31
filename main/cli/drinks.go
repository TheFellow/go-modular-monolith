package main

import (
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	drinkscli "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/cli"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	clitable "github.com/TheFellow/go-modular-monolith/main/cli/table"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/urfave/cli/v3"
)

func (c *CLI) drinksCommands() *cli.Command {
	return &cli.Command{
		Name:  "drinks",
		Usage: "Manage drinks",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List drinks",
				Flags: []cli.Flag{
					JSONFlag,
					&cli.StringFlag{Name: "name", Usage: "Filter by exact name match"},
					&cli.StringFlag{
						Name:    "category",
						Aliases: []string{"c"},
						Usage:   "Filter by category (e.g. cocktail, mocktail, tiki)",
						Validator: func(s string) error {
							return drinksmodels.DrinkCategory(strings.TrimSpace(s)).Validate()
						},
					},
					&cli.StringFlag{
						Name:    "glass",
						Aliases: []string{"g"},
						Usage:   "Filter by glass (e.g. coupe, rocks)",
						Validator: func(s string) error {
							return drinksmodels.GlassType(strings.TrimSpace(s)).Validate()
						},
					},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					res, err := c.app.Drinks.List(ctx, drinks.ListRequest{
						Name:     cmd.String("name"),
						Category: drinksmodels.DrinkCategory(cmd.String("category")),
						Glass:    drinksmodels.GlassType(cmd.String("glass")),
					})
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						out := make([]drinkscli.Drink, 0, len(res))
						for _, d := range res {
							out = append(out, drinkscli.FromDomainDrink(*d))
						}
						return writeJSON(cmd.Writer, out)
					}

					return clitable.PrintTable(drinkscli.ToDrinkRows(res))
				}),
			},
			{
				Name:  "get",
				Usage: "Get a drink by ID",
				Flags: []cli.Flag{
					JSONFlag,
					&cli.StringFlag{Name: "id", Usage: "Drink ID", Required: true},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					drinkID, err := entity.ParseDrinkID(cmd.String("id"))
					if err != nil {
						return err
					}
					res, err := c.app.Drinks.Get(ctx, drinkID)
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, drinkscli.FromDomainDrink(*res))
					}

					return clitable.PrintDetail(drinkscli.ToDrinkRow(res))
				}),
			},
			{
				Name:  "create",
				Usage: "Create a new drink",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "args", Max: 0},
				},
				Flags: []cli.Flag{
					TemplateFlag,
					StdinFlag,
					FileFlag,
					JSONFlag,
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					if cmd.Bool("template") {
						return writeJSON(cmd.Writer, drinkscli.TemplateCreateDrink())
					}

					created, err := readDrinkCreateInput(cmd)
					if err != nil {
						return err
					}

					res, err := c.app.Drinks.Create(ctx, &created)
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, drinkscli.FromDomainDrink(*res))
					}

					fmt.Println(res.ID.String())
					return nil
				}),
			},
			{
				Name:  "update",
				Usage: "Update a drink",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "args", Max: 0},
				},
				Flags: []cli.Flag{
					TemplateFlag,
					StdinFlag,
					FileFlag,
					JSONFlag,
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					if cmd.Bool("template") {
						return writeJSON(cmd.Writer, drinkscli.TemplateUpdateDrink())
					}

					updated, err := readDrinkUpdateInput(cmd)
					if err != nil {
						return err
					}

					res, err := c.app.Drinks.Update(ctx, &updated)
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, drinkscli.FromDomainDrink(*res))
					}

					fmt.Println(res.ID.String())
					return nil
				}),
			},
			{
				Name:  "delete",
				Usage: "Delete a drink by ID",
				Flags: []cli.Flag{
					JSONFlag,
					&cli.StringFlag{Name: "id", Usage: "Drink ID", Required: true},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					drinkID, err := entity.ParseDrinkID(cmd.String("id"))
					if err != nil {
						return err
					}
					res, err := c.app.Drinks.Delete(ctx, drinkID)
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, drinkscli.FromDomainDrink(*res))
					}

					fmt.Println(res.ID.String())
					return nil
				}),
			},
		},
	}
}

func readDrinkCreateInput(cmd *cli.Command) (drinksmodels.Drink, error) {
	input, err := readJSONInput[drinkscli.CreateDrink](cmd)
	if err != nil {
		return drinksmodels.Drink{}, err
	}
	return input.ToDomain()
}

func readDrinkUpdateInput(cmd *cli.Command) (drinksmodels.Drink, error) {
	input, err := readJSONInput[drinkscli.Drink](cmd)
	if err != nil {
		return drinksmodels.Drink{}, err
	}
	return input.ToDomainForUpdate()
}
