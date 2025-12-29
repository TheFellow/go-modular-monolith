package main

import (
	"context"
	"fmt"
	"os"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/drinks"
	"github.com/urfave/cli/v3"
)

func main() {
	var drinksDataPath string

	cmd := &cli.Command{
		Name:  "mixology",
		Usage: "Mixology as a Service",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "drinks-data",
				Usage:       "Path to drinks JSON data file",
				Value:       "pkg/data/drinks.json",
				Destination: &drinksDataPath,
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List drinks",
				Action: func(ctx context.Context, _ *cli.Command) error {
					a, err := app.New(app.WithDrinksDataPath(drinksDataPath))
					if err != nil {
						return err
					}

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
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		_, _ = fmt.Fprintln(cmd.ErrWriter, err)
		os.Exit(1)
	}
}
