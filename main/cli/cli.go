package main

import (
	"context"
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/urfave/cli/v3"
)

type CLI struct {
	app   *app.App
	actor string
}

func NewCLI() (*CLI, error) {
	if err := store.Open("data/mixology.db"); err != nil {
		return nil, err
	}
	if err := store.Register(context.Background(), app.StoreTypes()...); err != nil {
		return nil, err
	}
	return &CLI{app: app.New(), actor: "owner"}, nil
}

func (c *CLI) action(fn func(*middleware.Context, *cli.Command) error) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		mctx, ok := ctx.(*middleware.Context)
		if !ok {
			return apperrors.ToCLIExit(fmt.Errorf("expected middleware context"))
		}
		return apperrors.ToCLIExit(fn(mctx, cmd))
	}
}

func (c *CLI) Command() *cli.Command {
	return &cli.Command{
		Name:  "mixology",
		Usage: "Mixology as a Service",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "as",
				Usage:       "Actor to run as (owner|anonymous)",
				Value:       c.actor,
				Destination: &c.actor,
			},
		},
		Before: func(ctx context.Context, _ *cli.Command) (context.Context, error) {
			p, err := authn.ParseActor(c.actor)
			if err != nil {
				return ctx, err
			}
			return middleware.NewContext(ctx, middleware.WithPrincipal(p)), nil
		},
		ExitErrHandler: func(_ context.Context, _ *cli.Command, _ error) {},
		OnUsageError: func(_ context.Context, _ *cli.Command, err error, _ bool) error {
			return cli.Exit(err, apperrors.ExitUsage)
		},
		Commands: []*cli.Command{
			c.drinksCommands(),
			c.ingredientsCommands(),
			c.inventoryCommands(),
			c.menuCommands(),
			c.ordersCommands(),
		},
	}
}
