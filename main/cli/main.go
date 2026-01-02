package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/drinks"
	"github.com/TheFellow/go-modular-monolith/app/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/urfave/cli/v3"
)

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
			if a != nil {
				p, err := authn.ParseActor(actor)
				if err != nil {
					return ctx, err
				}
				return middleware.ContextWithPrincipal(ctx, p), nil
			}
			var err error
			a, err = app.New()
			if err != nil {
				return ctx, err
			}

			p, err := authn.ParseActor(actor)
			if err != nil {
				return ctx, err
			}
			return middleware.ContextWithPrincipal(ctx, p), nil
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
							res, err := a.Drinks().List(ctx, drinks.ListRequest{})
							if err != nil {
								return err
							}

							for _, d := range res.Drinks {
								fmt.Printf("%s\t%s\n", d.ID, d.Name)
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

							res, err := a.Drinks().Get(ctx, drinks.GetRequest{ID: id})
							if err != nil {
								return err
							}

							fmt.Printf("%s\t%s\n", res.Drink.ID, res.Drink.Name)
							return nil
						},
					},
					{
						Name:      "create",
						Usage:     "Create a new drink",
						ArgsUsage: "<name>",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							name := strings.TrimSpace(strings.Join(cmd.Args().Slice(), " "))
							if name == "" {
								return fmt.Errorf("missing name")
							}

							res, err := a.Drinks().Create(ctx, drinks.CreateRequest{Name: name})
							if err != nil {
								return err
							}

							fmt.Printf("%s\t%s\n", res.Drink.ID, res.Drink.Name)
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
							res, err := a.Ingredients().List(ctx, ingredients.ListRequest{})
							if err != nil {
								return err
							}

							for _, i := range res.Ingredients {
								fmt.Printf("%s\t%s\t%s\t%s\n", i.ID, i.Name, i.Category, i.Unit)
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

							res, err := a.Ingredients().Get(ctx, ingredients.GetRequest{ID: id})
							if err != nil {
								return err
							}

							i := res.Ingredient
							fmt.Printf("ID:          %s\n", i.ID)
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

							res, err := a.Ingredients().Create(ctx, ingredients.CreateRequest{
								Name:        name,
								Category:    models.Category(cmd.String("category")),
								Unit:        models.Unit(cmd.String("unit")),
								Description: cmd.String("description"),
							})
							if err != nil {
								return err
							}

							i := res.Ingredient
							fmt.Printf("%s\t%s\t%s\t%s\n", i.ID, i.Name, i.Category, i.Unit)
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

							res, err := a.Ingredients().Update(ctx, ingredients.UpdateRequest{
								ID:          id,
								Name:        cmd.String("name"),
								Category:    models.Category(cmd.String("category")),
								Unit:        models.Unit(cmd.String("unit")),
								Description: cmd.String("description"),
							})
							if err != nil {
								return err
							}

							i := res.Ingredient
							fmt.Printf("%s\t%s\t%s\t%s\n", i.ID, i.Name, i.Category, i.Unit)
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
