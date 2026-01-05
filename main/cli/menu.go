package main

import (
	"fmt"
	"strings"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	menuqueries "github.com/TheFellow/go-modular-monolith/app/domains/menu/queries"
	menucli "github.com/TheFellow/go-modular-monolith/app/domains/menu/surfaces/cli"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/urfave/cli/v3"
)

func (c *CLI) menuCommands() *cli.Command {
	return &cli.Command{
		Name:  "menu",
		Usage: "Curate drink menus",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List menus",
				Flags: []cli.Flag{JSONFlag, CostsFlag, TargetMarginFlag},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					res, err := c.app.Menu.List(ctx, menu.ListRequest{})
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						out := make([]menucli.Menu, 0, len(res.Menus))
						for _, m := range res.Menus {
							out = append(out, menucli.FromDomainMenu(m))
						}
						return writeJSON(cmd.Writer, out)
					}

					for _, m := range res.Menus {
						fmt.Printf("%s\t%s\t%s\t%d\n", string(m.ID.ID), m.Name, m.Status, len(m.Items))
						if cmd.Bool("costs") && len(m.Items) > 0 {
							an, err := menuqueries.NewAnalyticsCalculator().Analyze(ctx, m, cmd.Float64("target-margin"))
							if err != nil {
								return err
							}
							if an.AverageMargin != nil {
								fmt.Printf("\tavailable: %d/%d\tavg margin: %.0f%%\n", an.AvailableCount, an.TotalCount, *an.AverageMargin*100)
							} else {
								fmt.Printf("\tavailable: %d/%d\n", an.AvailableCount, an.TotalCount)
							}
						}
					}
					return nil
				}),
			},
			{
				Name:  "show",
				Usage: "Show a menu",
				Flags: []cli.Flag{JSONFlag, CostsFlag, TargetMarginFlag},
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "menu_id", UsageText: "Menu ID", Min: 1, Max: 1},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					id := cmd.StringArgs("menu_id")[0]
					res, err := c.app.Menu.Get(ctx, menu.GetRequest{ID: menumodels.NewMenuID(id)})
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						if cmd.Bool("costs") {
							an, err := menuqueries.NewAnalyticsCalculator().Analyze(ctx, res.Menu, cmd.Float64("target-margin"))
							if err != nil {
								return err
							}
							return writeJSON(cmd.Writer, an)
						}
						return writeJSON(cmd.Writer, menucli.FromDomainMenu(res.Menu))
					}

					m := res.Menu
					fmt.Printf("ID:          %s\n", string(m.ID.ID))
					fmt.Printf("Name:        %s\n", m.Name)
					if m.Description != "" {
						fmt.Printf("Description: %s\n", m.Description)
					}
					fmt.Printf("Status:      %s\n", m.Status)
					fmt.Printf("Items:\n")

					if cmd.Bool("costs") {
						an, err := menuqueries.NewAnalyticsCalculator().Analyze(ctx, m, cmd.Float64("target-margin"))
						if err != nil {
							return err
						}

						for _, item := range an.Items {
							parts := []string{fmt.Sprintf("- %s\t%s", string(item.DrinkID.ID), item.Name)}
							if item.Cost == nil || item.CostUnknown {
								parts = append(parts, "cost: n/a")
							} else {
								parts = append(parts, fmt.Sprintf("cost: %s", item.Cost.String()))
							}
							if item.MenuPrice != nil {
								parts = append(parts, fmt.Sprintf("price: %s", item.MenuPrice.String()))
								if item.Margin != nil {
									parts = append(parts, fmt.Sprintf("margin: %.0f%%", *item.Margin*100))
								}
							} else if item.SuggestedPrice != nil {
								parts = append(parts, fmt.Sprintf("suggested: %s", item.SuggestedPrice.String()))
							}

							status := string(item.Availability)
							if len(item.Substitutions) > 0 {
								sub := item.Substitutions[0]
								status = status + fmt.Sprintf(" (sub: %s for %s)", string(sub.Substitute.ID), string(sub.Original.ID))
							}
							parts = append(parts, fmt.Sprintf("[%s]", strings.ToUpper(status)))
							fmt.Println(strings.Join(parts, "\t"))
						}

						fmt.Printf("\nAnalytics:\n")
						fmt.Printf("  Available: %d/%d\n", an.AvailableCount, an.TotalCount)
						if an.AverageMargin != nil {
							fmt.Printf("  Average margin: %.0f%%\n", *an.AverageMargin*100)
						}
						return nil
					}

					for _, item := range m.Items {
						fmt.Printf("- %s\t%s\n", string(item.DrinkID.ID), item.Availability)
					}
					return nil
				}),
			},
			{
				Name:  "create",
				Usage: "Create a new menu",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "name", UsageText: "Menu name", Min: 1, Max: 1},
				},
				Flags: []cli.Flag{JSONFlag},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					name := cmd.StringArgs("name")[0]
					created, err := c.app.Menu.Create(ctx, menumodels.Menu{Name: name})
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, menucli.FromDomainMenu(created))
					}

					fmt.Printf("%s\t%s\n", string(created.ID.ID), created.Name)
					return nil
				}),
			},
			{
				Name:  "add-drink",
				Usage: "Add a drink to a menu",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "menu_id", UsageText: "Menu ID", Min: 1, Max: 1},
					&cli.StringArgs{Name: "drink_id", UsageText: "Drink ID", Min: 1, Max: 1},
				},
				Flags: []cli.Flag{JSONFlag},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					menuID := menumodels.NewMenuID(cmd.StringArgs("menu_id")[0])
					drinkID := drinksmodels.NewDrinkID(cmd.StringArgs("drink_id")[0])
					updated, err := c.app.Menu.AddDrink(ctx, menumodels.MenuDrinkChange{
						MenuID:  menuID,
						DrinkID: drinkID,
					})
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, menucli.FromDomainMenu(updated))
					}

					fmt.Printf("%s\t%s\t%d\n", string(updated.ID.ID), updated.Name, len(updated.Items))
					return nil
				}),
			},
			{
				Name:  "remove-drink",
				Usage: "Remove a drink from a menu",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "menu_id", UsageText: "Menu ID", Min: 1, Max: 1},
					&cli.StringArgs{Name: "drink_id", UsageText: "Drink ID", Min: 1, Max: 1},
				},
				Flags: []cli.Flag{JSONFlag},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					menuID := menumodels.NewMenuID(cmd.StringArgs("menu_id")[0])
					drinkID := drinksmodels.NewDrinkID(cmd.StringArgs("drink_id")[0])
					updated, err := c.app.Menu.RemoveDrink(ctx, menumodels.MenuDrinkChange{
						MenuID:  menuID,
						DrinkID: drinkID,
					})
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, menucli.FromDomainMenu(updated))
					}

					fmt.Printf("%s\t%s\t%d\n", string(updated.ID.ID), updated.Name, len(updated.Items))
					return nil
				}),
			},
			{
				Name:  "publish",
				Usage: "Publish a menu",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "menu_id", UsageText: "Menu ID", Min: 1, Max: 1},
				},
				Flags: []cli.Flag{JSONFlag},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					menuID := cmd.StringArgs("menu_id")[0]
					published, err := c.app.Menu.Publish(ctx, menumodels.Menu{ID: menumodels.NewMenuID(menuID)})
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, menucli.FromDomainMenu(published))
					}

					fmt.Printf("%s\t%s\t%s\n", string(published.ID.ID), published.Name, published.Status)
					return nil
				}),
			},
		},
	}
}
