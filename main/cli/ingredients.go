package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/urfave/cli/v3"
)

func ingredientsCommands(a *app.App) *cli.Command {
	return &cli.Command{
		Name:  "ingredients",
		Usage: "Manage ingredients",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List ingredients",
				Action: func(ctx context.Context, _ *cli.Command) error {
					mctx, err := requireMiddlewareContext(ctx)
					if err != nil {
						return err
					}
					if a == nil {
						return fmt.Errorf("app not initialized")
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

					mctx, err := requireMiddlewareContext(ctx)
					if err != nil {
						return err
					}
					if a == nil {
						return fmt.Errorf("app not initialized")
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

					mctx, err := requireMiddlewareContext(ctx)
					if err != nil {
						return err
					}
					if a == nil {
						return fmt.Errorf("app not initialized")
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

					mctx, err := requireMiddlewareContext(ctx)
					if err != nil {
						return err
					}
					if a == nil {
						return fmt.Errorf("app not initialized")
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
	}
}
