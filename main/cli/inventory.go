package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/inventory"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/urfave/cli/v3"
)

func inventoryCommands(a **app.App) *cli.Command {
	return &cli.Command{
		Name:  "inventory",
		Usage: "Manage ingredient stock",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List stock levels",
				Action: func(ctx context.Context, _ *cli.Command) error {
					mctx, err := requireMiddlewareContext(ctx)
					if err != nil {
						return err
					}
					if a == nil || *a == nil {
						return fmt.Errorf("app not initialized")
					}

					res, err := (*a).Inventory().List(mctx, inventory.ListRequest{})
					if err != nil {
						return err
					}

					for _, s := range res.Stock {
						fmt.Printf("%s\t%.2f\t%s\n", string(s.IngredientID.ID), s.Quantity, s.Unit)
					}
					return nil
				},
			},
			{
				Name:      "get",
				Usage:     "Get stock for an ingredient",
				ArgsUsage: "<ingredient-id>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					id := cmd.Args().First()
					if id == "" {
						return fmt.Errorf("missing ingredient-id")
					}

					mctx, err := requireMiddlewareContext(ctx)
					if err != nil {
						return err
					}
					if a == nil || *a == nil {
						return fmt.Errorf("app not initialized")
					}

					res, err := (*a).Inventory().Get(mctx, inventory.GetRequest{IngredientID: models.NewIngredientID(id)})
					if err != nil {
						return err
					}

					s := res.Stock
					fmt.Printf("%s\t%.2f\t%s\n", string(s.IngredientID.ID), s.Quantity, s.Unit)
					return nil
				},
			},
			{
				Name:      "adjust",
				Usage:     "Adjust stock by delta",
				ArgsUsage: "<ingredient-id> <delta>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "reason",
						Aliases:  []string{"r"},
						Usage:    "Reason (received|used|spilled|expired|corrected)",
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					ingredientID := cmd.Args().Get(0)
					deltaStr := cmd.Args().Get(1)
					if ingredientID == "" || deltaStr == "" {
						return fmt.Errorf("usage: inventory adjust <ingredient-id> <delta> --reason=<reason>")
					}

					delta, err := strconv.ParseFloat(deltaStr, 64)
					if err != nil {
						return fmt.Errorf("invalid delta: %w", err)
					}

					mctx, err := requireMiddlewareContext(ctx)
					if err != nil {
						return err
					}
					if a == nil || *a == nil {
						return fmt.Errorf("app not initialized")
					}

					res, err := (*a).Inventory().Adjust(mctx, inventory.AdjustRequest{
						IngredientID: models.NewIngredientID(ingredientID),
						Delta:        delta,
						Reason:       inventorymodels.AdjustmentReason(cmd.String("reason")),
					})
					if err != nil {
						return err
					}

					s := res.Stock
					fmt.Printf("%s\t%.2f\t%s\n", string(s.IngredientID.ID), s.Quantity, s.Unit)
					return nil
				},
			},
			{
				Name:      "set",
				Usage:     "Set stock quantity",
				ArgsUsage: "<ingredient-id> <quantity>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					ingredientID := cmd.Args().Get(0)
					qtyStr := cmd.Args().Get(1)
					if ingredientID == "" || qtyStr == "" {
						return fmt.Errorf("usage: inventory set <ingredient-id> <quantity>")
					}

					qty, err := strconv.ParseFloat(qtyStr, 64)
					if err != nil {
						return fmt.Errorf("invalid quantity: %w", err)
					}

					mctx, err := requireMiddlewareContext(ctx)
					if err != nil {
						return err
					}
					if a == nil || *a == nil {
						return fmt.Errorf("app not initialized")
					}

					res, err := (*a).Inventory().Set(mctx, inventory.SetRequest{
						IngredientID: models.NewIngredientID(ingredientID),
						Quantity:     qty,
					})
					if err != nil {
						return err
					}

					s := res.Stock
					fmt.Printf("%s\t%.2f\t%s\n", string(s.IngredientID.ID), s.Quantity, s.Unit)
					return nil
				},
			},
		},
	}
}
