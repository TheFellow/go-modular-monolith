package main

import (
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	menuqueries "github.com/TheFellow/go-modular-monolith/app/domains/menu/queries"
	menucli "github.com/TheFellow/go-modular-monolith/app/domains/menu/surfaces/cli"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
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
				Flags: []cli.Flag{
					JSONFlag,
					CostsFlag,
					TargetMarginFlag,
					&cli.StringFlag{
						Name:  "status",
						Usage: "Filter by status (draft|published|archived)",
						Validator: func(s string) error {
							s = strings.TrimSpace(s)
							if s == "" {
								return nil
							}
							return menumodels.MenuStatus(s).Validate()
						},
					},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					res, err := c.app.Menu.List(ctx, menu.ListRequest{
						Status: menumodels.MenuStatus(cmd.String("status")),
					})
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						out := make([]menucli.Menu, 0, len(res))
						for _, m := range res {
							out = append(out, menucli.FromDomainMenu(*m))
						}
						return writeJSON(cmd.Writer, out)
					}

					for _, m := range res {
						fmt.Printf("%s\t%s\t%s\t%d\n", m.ID.String(), m.Name, m.Status, len(m.Items))
						if cmd.Bool("costs") && len(m.Items) > 0 {
							an, err := menuqueries.NewAnalyticsCalculator().Analyze(ctx, *m, cmd.Float64("target-margin"))
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
				Flags: []cli.Flag{
					JSONFlag,
					CostsFlag,
					TargetMarginFlag,
					&cli.StringFlag{Name: "menu-id", Usage: "Menu ID", Required: true},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					menuID, err := entity.ParseMenuID(cmd.String("menu-id"))
					if err != nil {
						return err
					}
					res, err := c.app.Menu.Get(ctx, menuID)
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						if cmd.Bool("costs") {
							an, err := menuqueries.NewAnalyticsCalculator().Analyze(ctx, *res, cmd.Float64("target-margin"))
							if err != nil {
								return err
							}
							return writeJSON(cmd.Writer, an)
						}
						return writeJSON(cmd.Writer, menucli.FromDomainMenu(*res))
					}

					m := *res
					fmt.Printf("ID:          %s\n", m.ID.String())
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
							parts := []string{fmt.Sprintf("- %s\t%s", item.DrinkID.String(), item.Name)}
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
								status = status + fmt.Sprintf(" (sub: %s for %s)", sub.Substitute.String(), sub.Original.String())
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
						fmt.Printf("- %s\t%s\n", item.DrinkID.String(), item.Availability)
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
					created, err := c.app.Menu.Create(ctx, &menumodels.Menu{Name: name})
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, menucli.FromDomainMenu(*created))
					}

					fmt.Printf("%s\t%s\n", created.ID.String(), created.Name)
					return nil
				}),
			},
			{
				Name:  "add-drink",
				Usage: "Add a drink to a menu",
				Flags: []cli.Flag{
					JSONFlag,
					&cli.StringFlag{Name: "menu-id", Usage: "Menu ID", Required: true},
					&cli.StringFlag{Name: "drink-id", Usage: "Drink ID", Required: true},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					menuID, err := entity.ParseMenuID(cmd.String("menu-id"))
					if err != nil {
						return err
					}
					drinkID, err := entity.ParseDrinkID(cmd.String("drink-id"))
					if err != nil {
						return err
					}
					updated, err := c.app.Menu.AddDrink(ctx, &menumodels.MenuDrinkChange{
						MenuID:  menuID,
						DrinkID: drinkID,
					})
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, menucli.FromDomainMenu(*updated))
					}

					fmt.Printf("%s\t%s\t%d\n", updated.ID.String(), updated.Name, len(updated.Items))
					return nil
				}),
			},
			{
				Name:  "remove-drink",
				Usage: "Remove a drink from a menu",
				Flags: []cli.Flag{
					JSONFlag,
					&cli.StringFlag{Name: "menu-id", Usage: "Menu ID", Required: true},
					&cli.StringFlag{Name: "drink-id", Usage: "Drink ID", Required: true},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					menuID, err := entity.ParseMenuID(cmd.String("menu-id"))
					if err != nil {
						return err
					}
					drinkID, err := entity.ParseDrinkID(cmd.String("drink-id"))
					if err != nil {
						return err
					}
					updated, err := c.app.Menu.RemoveDrink(ctx, &menumodels.MenuDrinkChange{
						MenuID:  menuID,
						DrinkID: drinkID,
					})
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, menucli.FromDomainMenu(*updated))
					}

					fmt.Printf("%s\t%s\t%d\n", updated.ID.String(), updated.Name, len(updated.Items))
					return nil
				}),
			},
			{
				Name:  "publish",
				Usage: "Publish a menu",
				Flags: []cli.Flag{
					JSONFlag,
					&cli.StringFlag{Name: "menu-id", Usage: "Menu ID", Required: true},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					menuID, err := entity.ParseMenuID(cmd.String("menu-id"))
					if err != nil {
						return err
					}
					published, err := c.app.Menu.Publish(ctx, &menumodels.Menu{ID: menuID})
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, menucli.FromDomainMenu(*published))
					}

					fmt.Printf("%s\t%s\t%s\n", published.ID.String(), published.Name, published.Status)
					return nil
				}),
			},
		},
	}
}
