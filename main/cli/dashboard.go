package main

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/urfave/cli/v3"
)

func (c *CLI) dashboardCommand() *cli.Command {
	return &cli.Command{Name: "status", Usage: "Show the application dashboard aggregate", Flags: []cli.Flag{JSONFlag}, Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
		data, err := c.app.Dashboard(ctx)
		if err != nil {
			return err
		}
		if cmd.Bool("json") {
			return writeJSON(cmd.Writer, data)
		}
		if _, err = fmt.Fprintf(cmd.Writer, "DRINKS\tINGREDIENTS\tINVENTORY\tLOW_STOCK\tMENUS\tDRAFT_MENUS\tPUBLISHED_MENUS\tORDERS\tPENDING_ORDERS\tAUDIT\n%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n", data.DrinkCount, data.IngredientCount, data.InventoryCount, data.LowStockCount, data.MenuCount, data.DraftMenus, data.PublishedMenus, data.OrderCount, data.PendingOrders, data.AuditCount); err != nil {
			return err
		}
		if len(data.RecentActivity) == 0 {
			_, err = fmt.Fprintln(cmd.Writer, "\nRECENT ACTIVITY\n(none)")
			return err
		}
		if _, err = fmt.Fprintln(cmd.Writer, "\nRECENT ACTIVITY"); err != nil {
			return err
		}
		for _, item := range data.RecentActivity {
			if _, err = fmt.Fprintf(cmd.Writer, "%s\t%s\t%s\n", item.Timestamp.Format("2006-01-02T15:04:05Z07:00"), item.Actor, item.Action); err != nil {
				return err
			}
		}
		return nil
	})}
}
