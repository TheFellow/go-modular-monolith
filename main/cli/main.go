package main

import (
	"context"
	"fmt"
	"os"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/drinks"
	"github.com/urfave/cli/v3"
)

func buildApp() *cli.Command {
	var a *app.App

	cmd := &cli.Command{
		Name:  "mixology",
		Usage: "Mixology as a Service",
		Before: func(ctx context.Context, _ *cli.Command) (context.Context, error) {
			if a != nil {
				return ctx, nil
			}
			var err error
			a, err = app.New()
			return ctx, err
		},
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
