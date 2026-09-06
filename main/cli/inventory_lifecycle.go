package main

import (
	models "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	toolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/cli"
	"github.com/urfave/cli/v3"
)

func (c *CLI) disposalCommand() *cli.Command {
	return &cli.Command{Name: "dispose", Usage: "Record disposal of discontinued or quarantined physical stock", Flags: []cli.Flag{&cli.StringFlag{Name: "ingredient-id", Required: true}, &cli.Float64Flag{Name: "quantity", Required: true}, &cli.StringFlag{Name: "reason", Required: true}, &cli.Uint64Flag{Name: "revision"}}, Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
		id, err := entity.ParseIngredientID(cmd.String("ingredient-id"))
		if err != nil {
			return err
		}
		stock, err := c.app.Inventory.Get(ctx, id)
		if err != nil {
			return err
		}
		amount, err := measurement.NewAmount(cmd.Float64("quantity"), stock.Amount.Unit())
		if err != nil {
			return err
		}
		revision := stock.Revision
		if cmd.IsSet("revision") {
			revision = cmd.Uint64("revision")
		}
		result, err := c.app.Inventory.Dispose(ctx, models.Disposal{IngredientID: id, Revision: revision, Amount: amount, Reason: cmd.String("reason")})
		if err != nil {
			return err
		}
		return toolkit.WriteJSON(cmd.Writer, result)
	})}
}
func (c *CLI) stockHistoryCommand() *cli.Command {
	return &cli.Command{Name: "history", Usage: "Show retained physical stock movements", Flags: []cli.Flag{&cli.StringFlag{Name: "ingredient-id", Required: true}}, Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
		id, err := entity.ParseIngredientID(cmd.String("ingredient-id"))
		if err != nil {
			return err
		}
		history, err := c.app.Inventory.History(ctx, id)
		if err != nil {
			return err
		}
		return toolkit.WriteJSON(cmd.Writer, history)
	})}
}

func (c *CLI) dispositionCommand(name string, quarantine bool) *cli.Command {
	return &cli.Command{Name: name, Usage: "Change retained stock disposition with a recorded reason", Flags: []cli.Flag{&cli.StringFlag{Name: "ingredient-id", Required: true}, &cli.StringFlag{Name: "reason", Required: true}, &cli.Uint64Flag{Name: "revision"}}, Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
		id, err := entity.ParseIngredientID(cmd.String("ingredient-id"))
		if err != nil {
			return err
		}
		stock, err := c.app.Inventory.Get(ctx, id)
		if err != nil {
			return err
		}
		revision := stock.Revision
		if cmd.IsSet("revision") {
			revision = cmd.Uint64("revision")
		}
		result, err := c.app.Inventory.Disposition(ctx, models.Disposition{IngredientID: id, Revision: revision, Quarantine: quarantine, Reason: cmd.String("reason")})
		if err != nil {
			return err
		}
		return toolkit.WriteJSON(cmd.Writer, result)
	})}
}
