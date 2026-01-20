package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	orderscli "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces/cli"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	clitable "github.com/TheFellow/go-modular-monolith/main/cli/table"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/urfave/cli/v3"
)

func (c *CLI) ordersCommands() *cli.Command {
	return &cli.Command{
		Name:  "order",
		Usage: "Manage orders",
		Commands: []*cli.Command{
			{
				Name:  "place",
				Usage: "Place an order",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "items", UsageText: "<drink-id>:<qty> [<drink-id>:<qty>...]", Max: 0},
				},
				Flags: []cli.Flag{
					JSONFlag,
					TemplateFlag,
					StdinFlag,
					FileFlag,
					&cli.StringFlag{Name: "menu-id", Usage: "Menu ID"},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					if cmd.Bool("template") {
						return writeJSON(cmd.Writer, orderscli.TemplatePlace())
					}

					var input *ordersmodels.Order
					if cmd.Bool("stdin") || strings.TrimSpace(cmd.String("file")) != "" {
						doc, err := readJSONInput[orderscli.OrderInput](cmd)
						if err != nil {
							return err
						}
						parsed, err := doc.ToDomain()
						if err != nil {
							return err
						}
						input = parsed
					} else {
						menuIDRaw := strings.TrimSpace(cmd.String("menu-id"))
						if menuIDRaw == "" {
							return errors.Invalidf("menu-id is required (or use --stdin/--file)")
						}
						menuID, err := entity.ParseMenuID(menuIDRaw)
						if err != nil {
							return err
						}

						args := cmd.StringArgs("items")
						if len(args) == 0 {
							return errors.Invalidf("items are required (or use --stdin/--file)")
						}
						items := make([]ordersmodels.OrderItem, 0, len(args))
						for _, spec := range args {
							parts := strings.SplitN(spec, ":", 2)
							if len(parts) != 2 {
								return fmt.Errorf("invalid item %q (expected drink-id:qty)", spec)
							}
							qty, err := strconv.Atoi(parts[1])
							if err != nil || qty <= 0 {
								return fmt.Errorf("invalid quantity in %q", spec)
							}
							drinkID, err := entity.ParseDrinkID(parts[0])
							if err != nil {
								return err
							}
							items = append(items, ordersmodels.OrderItem{
								DrinkID:  drinkID,
								Quantity: qty,
							})
						}
						input = &ordersmodels.Order{
							MenuID: menuID,
							Items:  items,
						}
					}

					created, err := c.app.Orders.Place(ctx, input)
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, created)
					}

					fmt.Println(created.ID.String())
					return nil
				}),
			},
			{
				Name:  "list",
				Usage: "List orders",
				Flags: []cli.Flag{
					JSONFlag,
					&cli.StringFlag{
						Name:  "status",
						Usage: "Filter by status (pending|preparing|completed|cancelled)",
						Validator: func(s string) error {
							s = strings.TrimSpace(s)
							if s == "" {
								return nil
							}
							return ordersmodels.OrderStatus(s).Validate()
						},
					},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					res, err := c.app.Orders.List(ctx, orders.ListRequest{
						Status: ordersmodels.OrderStatus(cmd.String("status")),
					})
					if err != nil {
						return err
					}
					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, res)
					}
					return clitable.PrintTable(orderscli.ToOrderRows(res))
				}),
			},
			{
				Name:  "get",
				Usage: "Get an order",
				Flags: []cli.Flag{
					JSONFlag,
					&cli.StringFlag{Name: "id", Usage: "Order ID", Required: true},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					orderID, err := entity.ParseOrderID(cmd.String("id"))
					if err != nil {
						return err
					}
					res, err := c.app.Orders.Get(ctx, orderID)
					if err != nil {
						return err
					}
					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, res)
					}
					if err := clitable.PrintDetail(orderscli.ToOrderDetail(res)); err != nil {
						return err
					}

					fmt.Println()
					return clitable.PrintTable(orderscli.ToOrderItemRows(res.Items))
				}),
			},
			{
				Name:  "complete",
				Usage: "Complete an order",
				Flags: []cli.Flag{
					JSONFlag,
					&cli.StringFlag{Name: "id", Usage: "Order ID", Required: true},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					orderID, err := entity.ParseOrderID(cmd.String("id"))
					if err != nil {
						return err
					}
					updated, err := c.app.Orders.Complete(ctx, &ordersmodels.Order{ID: orderID})
					if err != nil {
						return err
					}
					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, updated)
					}
					fmt.Println(updated.ID.String())
					return nil
				}),
			},
			{
				Name:  "cancel",
				Usage: "Cancel an order",
				Flags: []cli.Flag{
					JSONFlag,
					&cli.StringFlag{Name: "id", Usage: "Order ID", Required: true},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					orderID, err := entity.ParseOrderID(cmd.String("id"))
					if err != nil {
						return err
					}
					updated, err := c.app.Orders.Cancel(ctx, &ordersmodels.Order{ID: orderID})
					if err != nil {
						return err
					}
					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, updated)
					}
					fmt.Println(updated.ID.String())
					return nil
				}),
			},
		},
	}
}
