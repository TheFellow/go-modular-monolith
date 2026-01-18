package main

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/urfave/cli/v3"
)

func (c *CLI) inventoryCommands() *cli.Command {
	return &cli.Command{
		Name:  "inventory",
		Usage: "Manage ingredient stock",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List stock levels",
				Flags: []cli.Flag{
					&cli.Float64Flag{
						Name:  "low-stock",
						Usage: "Show items with quantity <= threshold",
						Value: -1,
					},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					var req inventory.ListRequest
					if v := cmd.Float64("low-stock"); v >= 0 {
						req.LowStock = optional.Some(v)
					}
					res, err := c.app.Inventory.List(ctx, req)
					if err != nil {
						return err
					}

					for _, s := range res {
						fmt.Printf("%s\t%.2f\t%s\n", s.IngredientID.String(), s.Quantity, s.Unit)
					}
					return nil
				}),
			},
			{
				Name:  "get",
				Usage: "Get stock for an ingredient",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "ingredient_id", UsageText: "Ingredient ID", Min: 1, Max: 1},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					id := cmd.StringArgs("ingredient_id")[0]
					ingredientID, err := parseIngredientID(id)
					if err != nil {
						return err
					}
					res, err := c.app.Inventory.Get(ctx, ingredientID)
					if err != nil {
						return err
					}

					s := res
					fmt.Printf("%s\t%.2f\t%s\n", s.IngredientID.String(), s.Quantity, s.Unit)
					return nil
				}),
			},
			{
				Name:  "adjust",
				Usage: "Patch stock quantity and/or cost",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "ingredient_id", UsageText: "Ingredient ID", Min: 1, Max: 1},
					&cli.Float64Args{Name: "delta", UsageText: "Delta (+/-)", Min: 0, Max: 1},
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "reason",
						Aliases:  []string{"r"},
						Usage:    "Reason (received|used|spilled|expired|corrected)",
						Required: true,
						Validator: func(s string) error {
							switch inventorymodels.AdjustmentReason(s) {
							case inventorymodels.ReasonReceived, inventorymodels.ReasonUsed, inventorymodels.ReasonSpilled, inventorymodels.ReasonExpired, inventorymodels.ReasonCorrected:
								return nil
							default:
								return fmt.Errorf("invalid reason: %s", s)
							}
						},
					},
					&cli.StringFlag{
						Name:  "cost-per-unit",
						Usage: "Cost per unit (e.g. \"$1.23\" or \"USD 1.23\")",
					},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					ingredientID := cmd.StringArgs("ingredient_id")[0]
					deltaArgs := cmd.Float64Args("delta")
					parsedIngredientID, err := parseIngredientID(ingredientID)
					if err != nil {
						return err
					}

					var delta optional.Value[float64]
					if len(deltaArgs) == 1 {
						delta = optional.Some(deltaArgs[0])
					}

					var cost optional.Value[money.Price]
					if s := cmd.String("cost-per-unit"); s != "" {
						p, err := parsePrice(s)
						if err != nil {
							return err
						}
						cost = optional.Some(p)
					}

					res, err := c.app.Inventory.Adjust(ctx, &inventorymodels.Patch{
						IngredientID: parsedIngredientID,
						Delta:        delta,
						CostPerUnit:  cost,
						Reason:       inventorymodels.AdjustmentReason(cmd.String("reason")),
					})
					if err != nil {
						return err
					}

					fmt.Printf("%s\t%.2f\t%s\n", res.IngredientID.String(), res.Quantity, res.Unit)
					return nil
				}),
			},
			{
				Name:  "set",
				Usage: "Set stock quantity",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "ingredient_id", UsageText: "Ingredient ID", Min: 1, Max: 1},
					&cli.Float64Args{Name: "quantity", UsageText: "Quantity", Min: 1, Max: 1},
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "cost-per-unit",
						Usage:    "Cost per unit (e.g. \"$1.23\" or \"USD 1.23\")",
						Required: true,
					},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					ingredientID := cmd.StringArgs("ingredient_id")[0]
					qty := cmd.Float64Args("quantity")[0]
					parsedIngredientID, err := parseIngredientID(ingredientID)
					if err != nil {
						return err
					}

					cost, err := parsePrice(cmd.String("cost-per-unit"))
					if err != nil {
						return err
					}

					res, err := c.app.Inventory.Set(ctx, &inventorymodels.Update{
						IngredientID: parsedIngredientID,
						Quantity:     qty,
						CostPerUnit:  cost,
					})
					if err != nil {
						return err
					}

					fmt.Printf("%s\t%.2f\t%s\n", res.IngredientID.String(), res.Quantity, res.Unit)
					return nil
				}),
			},
		},
	}
}
