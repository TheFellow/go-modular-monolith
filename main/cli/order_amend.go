package main

import (
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	orderscli "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces/cli"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	toolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/cli"
	"github.com/urfave/cli/v3"
)

func (c *CLI) orderAmendCommand() *cli.Command {
	return &cli.Command{Name: "amend", Usage: "Approve an ingredient replacement without changing original order acceptance", Flags: []cli.Flag{&cli.StringFlag{Name: "id", Required: true}, &cli.StringFlag{Name: "ingredient-id", Required: true}, &cli.StringFlag{Name: "replacement-id", Required: true}, &cli.Float64Flag{Name: "ratio", Value: 1}, &cli.StringFlag{Name: "reason", Required: true}, &cli.Uint64Flag{Name: "revision", Usage: "Expected order revision (defaults to the currently loaded order)"}, toolkit.JSONFlag}, Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
		id, err := entity.ParseOrderID(cmd.String("id"))
		if err != nil {
			return err
		}
		original, err := entity.ParseIngredientID(cmd.String("ingredient-id"))
		if err != nil {
			return err
		}
		replacement, err := entity.ParseIngredientID(cmd.String("replacement-id"))
		if err != nil {
			return err
		}
		order, err := c.app.Orders.Get(ctx, id)
		if err != nil {
			return err
		}
		revision := order.Revision
		if cmd.IsSet("revision") {
			revision = cmd.Uint64("revision")
		}
		result, err := c.app.Orders.Amend(ctx, ordersmodels.Amendment{OrderID: id, Revision: revision, Reason: cmd.String("reason"), Replacements: []ordersmodels.Replacement{{OriginalID: original, ReplacementID: replacement, Ratio: cmd.Float64("ratio")}}})
		if err != nil {
			return err
		}
		return toolkit.WriteJSON(cmd.Writer, orderscli.ToOrderView(result))
	})}
}

func (c *CLI) orderAmendBatchCommand() *cli.Command {
	return &cli.Command{Name: "amend-batch", Usage: "Atomically approve a JSON array of order amendments with expected revisions", Flags: []cli.Flag{toolkit.FileFlag, toolkit.StdinFlag}, Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
		requests, err := toolkit.ReadJSONInput[[]ordersmodels.Amendment](cmd)
		if err != nil {
			return err
		}
		result, err := c.app.AmendOrders(ctx, requests)
		if err != nil {
			return err
		}
		views := make([]orderscli.OrderView, 0, len(result))
		for _, order := range result {
			views = append(views, orderscli.ToOrderView(order))
		}
		return toolkit.WriteJSON(cmd.Writer, views)
	})}
}
