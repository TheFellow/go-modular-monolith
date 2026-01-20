package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	inventorycli "github.com/TheFellow/go-modular-monolith/app/domains/inventory/surfaces/cli"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	clitable "github.com/TheFellow/go-modular-monolith/main/cli/table"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
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
					JSONFlag,
					&cli.Float64Flag{
						Name:  "low-stock",
						Usage: "Show items with amount <= threshold (per item unit)",
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

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, res)
					}

					return clitable.PrintTable(inventorycli.ToInventoryRows(res))
				}),
			},
			{
				Name:  "get",
				Usage: "Get stock for an ingredient",
				Flags: []cli.Flag{
					JSONFlag,
					&cli.StringFlag{Name: "ingredient-id", Usage: "Ingredient ID", Required: true},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					ingredientID, err := entity.ParseIngredientID(cmd.String("ingredient-id"))
					if err != nil {
						return err
					}
					res, err := c.app.Inventory.Get(ctx, ingredientID)
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, res)
					}

					return clitable.PrintDetail(inventorycli.ToInventoryRow(res))
				}),
			},
			{
				Name:  "adjust",
				Usage: "Patch stock quantity and/or cost",
				Flags: []cli.Flag{
					JSONFlag,
					&cli.StringFlag{Name: "ingredient-id", Usage: "Ingredient ID", Required: true},
					&cli.StringFlag{Name: "delta", Usage: "Delta (+/-) in ingredient unit"},
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
						Usage: "Cost per unit in ingredient unit (e.g. \"$1.23\" or \"USD 1.23\")",
					},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					parsedIngredientID, err := entity.ParseIngredientID(cmd.String("ingredient-id"))
					if err != nil {
						return err
					}
					ingredient, err := c.app.Ingredients.Get(ctx, parsedIngredientID)
					if err != nil {
						return err
					}

					var delta optional.Value[measurement.Amount]
					if raw := strings.TrimSpace(cmd.String("delta")); raw != "" {
						v, err := strconv.ParseFloat(raw, 64)
						if err != nil {
							return errors.Invalidf("invalid delta %q", raw)
						}
						amount, err := measurement.NewAmount(v, ingredient.Unit)
						if err != nil {
							return err
						}
						delta = optional.Some(amount)
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

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, res)
					}

					fmt.Println(res.IngredientID.String())
					return nil
				}),
			},
			{
				Name:  "set",
				Usage: "Set stock quantity",
				Flags: []cli.Flag{
					JSONFlag,
					&cli.StringFlag{Name: "ingredient-id", Usage: "Ingredient ID", Required: true},
					&cli.Float64Flag{Name: "quantity", Usage: "Quantity in ingredient unit", Required: true},
					&cli.StringFlag{
						Name:     "cost-per-unit",
						Usage:    "Cost per unit in ingredient unit (e.g. \"$1.23\" or \"USD 1.23\")",
						Required: true,
					},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					parsedIngredientID, err := entity.ParseIngredientID(cmd.String("ingredient-id"))
					if err != nil {
						return err
					}
					ingredient, err := c.app.Ingredients.Get(ctx, parsedIngredientID)
					if err != nil {
						return err
					}
					qty := cmd.Float64("quantity")

					cost, err := parsePrice(cmd.String("cost-per-unit"))
					if err != nil {
						return err
					}
					amount, err := measurement.NewAmount(qty, ingredient.Unit)
					if err != nil {
						return err
					}

					res, err := c.app.Inventory.Set(ctx, &inventorymodels.Update{
						IngredientID: parsedIngredientID,
						Amount:       amount,
						CostPerUnit:  cost,
					})
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, res)
					}

					fmt.Println(res.IngredientID.String())
					return nil
				}),
			},
		},
	}
}
