package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
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
					&cli.StringArgs{Name: "args", UsageText: "<menu-id> <drink-id>:<qty> [<drink-id>:<qty>...]", Min: 2, Max: 0},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					args := cmd.StringArgs("args")
					menuID := menumodels.NewMenuID(args[0])

					items := make([]ordersmodels.OrderItem, 0, len(args)-1)
					for _, spec := range args[1:] {
						parts := strings.SplitN(spec, ":", 2)
						if len(parts) != 2 {
							return fmt.Errorf("invalid item %q (expected drink-id:qty)", spec)
						}
						qty, err := strconv.Atoi(parts[1])
						if err != nil || qty <= 0 {
							return fmt.Errorf("invalid quantity in %q", spec)
						}
						items = append(items, ordersmodels.OrderItem{
							DrinkID:  drinksmodels.NewDrinkID(parts[0]),
							Quantity: qty,
						})
					}

					created, err := c.app.Orders.Place(ctx, ordersmodels.Order{
						ID:     ordersmodels.NewOrderID(""),
						MenuID: menuID,
						Items:  items,
					})
					if err != nil {
						return err
					}

					fmt.Printf("%s\t%s\t%s\t%d\n", string(created.ID.ID), string(created.MenuID.ID), created.Status, len(created.Items))
					return nil
				}),
			},
			{
				Name:  "list",
				Usage: "List orders",
				Flags: []cli.Flag{
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
					for _, o := range res.Orders {
						fmt.Printf("%s\t%s\t%s\t%s\n", string(o.ID.ID), string(o.MenuID.ID), o.Status, o.CreatedAt.Format(time.RFC3339))
					}
					return nil
				}),
			},
			{
				Name:  "get",
				Usage: "Get an order",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "order_id", UsageText: "Order ID", Min: 1, Max: 1},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					id := cmd.StringArgs("order_id")[0]
					res, err := c.app.Orders.Get(ctx, orders.GetRequest{ID: ordersmodels.NewOrderID(id)})
					if err != nil {
						return err
					}
					o := res.Order
					fmt.Printf("ID:        %s\n", string(o.ID.ID))
					fmt.Printf("MenuID:    %s\n", string(o.MenuID.ID))
					fmt.Printf("Status:    %s\n", o.Status)
					fmt.Printf("CreatedAt: %s\n", o.CreatedAt.Format(time.RFC3339))
					if t, ok := o.CompletedAt.Unwrap(); ok {
						fmt.Printf("CompletedAt: %s\n", t.Format(time.RFC3339))
					}
					if o.Notes != "" {
						fmt.Printf("Notes: %s\n", o.Notes)
					}
					fmt.Printf("Items:\n")
					for _, it := range o.Items {
						fmt.Printf("- %s\t%d\n", string(it.DrinkID.ID), it.Quantity)
					}
					return nil
				}),
			},
			{
				Name:  "complete",
				Usage: "Complete an order",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "order_id", UsageText: "Order ID", Min: 1, Max: 1},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					id := cmd.StringArgs("order_id")[0]
					updated, err := c.app.Orders.Complete(ctx, ordersmodels.Order{ID: ordersmodels.NewOrderID(id)})
					if err != nil {
						return err
					}
					fmt.Printf("%s\t%s\n", string(updated.ID.ID), updated.Status)
					return nil
				}),
			},
			{
				Name:  "cancel",
				Usage: "Cancel an order",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "order_id", UsageText: "Order ID", Min: 1, Max: 1},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					id := cmd.StringArgs("order_id")[0]
					updated, err := c.app.Orders.Cancel(ctx, ordersmodels.Order{ID: ordersmodels.NewOrderID(id)})
					if err != nil {
						return err
					}
					fmt.Printf("%s\t%s\n", string(updated.ID.ID), updated.Status)
					return nil
				}),
			},
		},
	}
}
