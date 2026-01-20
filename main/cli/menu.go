package main

import (
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	menuqueries "github.com/TheFellow/go-modular-monolith/app/domains/menu/queries"
	menucli "github.com/TheFellow/go-modular-monolith/app/domains/menu/surfaces/cli"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	clitable "github.com/TheFellow/go-modular-monolith/main/cli/table"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
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

					rows := make([]menucli.MenuRow, 0, len(res))
					for _, m := range res {
						rows = append(rows, menucli.ToMenuRow(m))
						if cmd.Bool("costs") && len(m.Items) > 0 {
							an, err := menuqueries.NewAnalyticsCalculator().Analyze(ctx, *m, cmd.Float64("target-margin"))
							if err != nil {
								return err
							}
							if an.AverageMargin != nil {
								rows = append(rows, menucli.MenuRow{
									Name:   fmt.Sprintf("available: %d/%d", an.AvailableCount, an.TotalCount),
									Status: fmt.Sprintf("avg margin: %.0f%%", *an.AverageMargin*100),
								})
							} else {
								rows = append(rows, menucli.MenuRow{
									Name: fmt.Sprintf("available: %d/%d", an.AvailableCount, an.TotalCount),
								})
							}
						}
					}
					return clitable.PrintTable(rows)
				}),
			},
			{
				Name:  "show",
				Usage: "Show a menu",
				Flags: []cli.Flag{
					JSONFlag,
					CostsFlag,
					TargetMarginFlag,
					&cli.StringFlag{Name: "id", Usage: "Menu ID", Required: true},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					menuID, err := entity.ParseMenuID(cmd.String("id"))
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
					if err := clitable.PrintDetail(menucli.ToMenuRow(&m)); err != nil {
						return err
					}

					if cmd.Bool("costs") {
						an, err := menuqueries.NewAnalyticsCalculator().Analyze(ctx, m, cmd.Float64("target-margin"))
						if err != nil {
							return err
						}

						if len(an.Items) > 0 {
							fmt.Println()
							w := newTabWriter()
							fmt.Fprintln(w, "DRINK_ID\tNAME\tCOST\tPRICE\tMARGIN\tSTATUS")
							for _, item := range an.Items {
								cost := "n/a"
								if item.Cost != nil && !item.CostUnknown {
									cost = item.Cost.String()
								}

								price := "n/a"
								if item.MenuPrice != nil {
									price = item.MenuPrice.String()
								} else if item.SuggestedPrice != nil {
									price = "suggested " + item.SuggestedPrice.String()
								}

								margin := "n/a"
								if item.Margin != nil {
									margin = fmt.Sprintf("%.0f%%", *item.Margin*100)
								}

								status := string(item.Availability)
								if len(item.Substitutions) > 0 {
									sub := item.Substitutions[0]
									status = status + fmt.Sprintf(" (sub: %s for %s)", sub.Substitute.String(), sub.Original.String())
								}

								fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", item.DrinkID.String(), item.Name, cost, price, margin, strings.ToUpper(status))
							}
							if err := w.Flush(); err != nil {
								return err
							}
						}

						fmt.Println()
						fmt.Println("Analytics:")
						w := newTabWriter()
						fmt.Fprintf(w, "Available:\t%d/%d\n", an.AvailableCount, an.TotalCount)
						if an.AverageMargin != nil {
							fmt.Fprintf(w, "Average margin:\t%.0f%%\n", *an.AverageMargin*100)
						}
						return w.Flush()
					}

					if len(m.Items) == 0 {
						return nil
					}

					fmt.Println()
					return clitable.PrintTable(menucli.ToMenuItemRows(m.Items))
				}),
			},
			{
				Name:  "create",
				Usage: "Create a new menu",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "name", UsageText: "Menu name", Max: 1},
				},
				Flags: []cli.Flag{
					JSONFlag,
					TemplateFlag,
					StdinFlag,
					FileFlag,
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					if cmd.Bool("template") {
						return writeJSON(cmd.Writer, menucli.TemplateCreate())
					}

					var input *menumodels.Menu
					if cmd.Bool("stdin") || strings.TrimSpace(cmd.String("file")) != "" {
						row, err := readJSONInput[menucli.MenuRow](cmd)
						if err != nil {
							return err
						}
						input = &menumodels.Menu{
							Name:        row.Name,
							Description: row.Desc,
						}
					} else {
						args := cmd.StringArgs("name")
						if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
							return errors.Invalidf("name is required (or use --stdin/--file)")
						}
						input = &menumodels.Menu{Name: args[0]}
					}

					created, err := c.app.Menu.Create(ctx, input)
					if err != nil {
						return err
					}

					if cmd.Bool("json") {
						return writeJSON(cmd.Writer, menucli.FromDomainMenu(*created))
					}

					fmt.Println(created.ID.String())
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

					fmt.Println(updated.ID.String())
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

					fmt.Println(updated.ID.String())
					return nil
				}),
			},
			{
				Name:  "publish",
				Usage: "Publish a menu",
				Flags: []cli.Flag{
					JSONFlag,
					&cli.StringFlag{Name: "id", Usage: "Menu ID", Required: true},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					menuID, err := entity.ParseMenuID(cmd.String("id"))
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

					fmt.Println(published.ID.String())
					return nil
				}),
			},
		},
	}
}
