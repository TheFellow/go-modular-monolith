package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/drinks"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/drinks/models"
	drinkscli "github.com/TheFellow/go-modular-monolith/app/drinks/surfaces/cli"
	"github.com/urfave/cli/v3"
)

func drinksCommands(a *app.App) *cli.Command {
	return &cli.Command{
		Name:  "drinks",
		Usage: "Manage drinks",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List drinks",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "json",
						Usage: "Output JSON",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					mctx, err := requireMiddlewareContext(ctx)
					if err != nil {
						return err
					}
					if a == nil {
						return fmt.Errorf("app not initialized")
					}

					res, err := a.Drinks().List(mctx, drinks.ListRequest{})
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						out := make([]drinkscli.Drink, 0, len(res.Drinks))
						for _, d := range res.Drinks {
							out = append(out, drinkscli.FromDomainDrink(d))
						}
						return writeJSON(cmd.Writer, out)
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
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "json",
						Usage: "Output JSON",
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

					res, err := a.Drinks().Get(mctx, drinks.GetRequest{ID: drinksmodels.NewDrinkID(id)})
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, drinkscli.FromDomainDrink(res.Drink))
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
					&cli.BoolFlag{
						Name:  "template",
						Usage: "Print recipe JSON template and exit",
					},
					&cli.BoolFlag{
						Name:  "stdin",
						Usage: "Read recipe JSON from stdin",
					},
					&cli.StringFlag{
						Name:  "file",
						Usage: "Read recipe JSON from file",
					},
					&cli.StringFlag{
						Name:  "category",
						Usage: "Category (cocktail|mocktail|shot|highball|martini|sour|tiki)",
					},
					&cli.StringFlag{
						Name:  "glass",
						Usage: "Glass (rocks|highball|coupe|martini)",
					},
					&cli.StringFlag{
						Name:  "description",
						Usage: "Description",
					},
					&cli.BoolFlag{
						Name:  "json",
						Usage: "Output JSON",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.Bool("template") {
						return writeJSON(cmd.Writer, drinkscli.TemplateRecipe())
					}

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

					recipe, err := readRecipeInput(cmd)
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

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, drinkscli.FromDomainDrink(res.Drink))
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
					&cli.BoolFlag{
						Name:  "template",
						Usage: "Print recipe JSON template and exit",
					},
					&cli.BoolFlag{
						Name:  "stdin",
						Usage: "Read recipe JSON from stdin",
					},
					&cli.StringFlag{
						Name:  "file",
						Usage: "Read recipe JSON from file",
					},
					&cli.BoolFlag{
						Name:  "json",
						Usage: "Output JSON",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.Bool("template") {
						return writeJSON(cmd.Writer, drinkscli.TemplateRecipe())
					}

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

					recipe, err := readRecipeInput(cmd)
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

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, drinkscli.FromDomainDrink(res.Drink))
					}

					fmt.Printf("%s\t%s\n", string(res.Drink.ID.ID), res.Drink.Name)
					return nil
				},
			},
		},
	}
}

func readRecipeInput(cmd *cli.Command) (drinksmodels.Recipe, error) {
	fromStdin := cmd.Bool("stdin")
	fromFile := strings.TrimSpace(cmd.String("file"))
	if fromStdin && fromFile != "" {
		return drinksmodels.Recipe{}, fmt.Errorf("set only one of --stdin or --file")
	}
	if !fromStdin && fromFile == "" {
		return drinksmodels.Recipe{}, fmt.Errorf("missing recipe input: set --stdin or --file (or use --template)")
	}

	var r io.Reader
	if fromStdin {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return drinksmodels.Recipe{}, err
		}
		if len(bytes.TrimSpace(b)) == 0 {
			return drinksmodels.Recipe{}, fmt.Errorf("stdin is empty")
		}
		r = bytes.NewReader(b)
	} else {
		f, err := os.Open(fromFile)
		if err != nil {
			return drinksmodels.Recipe{}, err
		}
		defer f.Close()
		r = f
	}

	return drinkscli.DecodeRecipeJSON(r)
}
