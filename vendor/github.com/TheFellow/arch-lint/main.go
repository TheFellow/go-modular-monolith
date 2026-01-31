package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/TheFellow/arch-lint/pkg/config"
	"github.com/TheFellow/arch-lint/pkg/linter"
)

func main() {
	app := &cli.Command{
		Name:  "arch-lint",
		Usage: "Enforce Go project architecture rules",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Value: "config.yaml", Usage: "Path to config file"},
			&cli.BoolFlag{Name: "verbose", Aliases: []string{"v"}, Usage: "Enable verbose output"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			cfg, err := config.Load(c.String("config"))
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			violations, err := linter.Run(cfg)
			if c.Bool("verbose") {
				fmt.Println(linter.Processed.String())
			}
			if err != nil {
				return err
			}

			if len(violations) > 0 {
				slices.SortFunc(violations, func(a, b linter.Violation) int {
					return strings.Compare(a.String(), b.String())
				})
				for _, v := range violations {
					fmt.Println(v)
				}
				os.Exit(1)
			}
			fmt.Println("✔ arch-lint: no forbidden imports found.")
			return nil
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
