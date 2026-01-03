package main

import (
	"context"
	"fmt"
	"os"

	"github.com/TheFellow/go-modular-monolith/app"
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
			drinksCommands(&a),
			ingredientsCommands(&a),
			inventoryCommands(&a),
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
