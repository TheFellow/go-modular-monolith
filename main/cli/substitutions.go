package main

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	toolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/cli"
	"github.com/urfave/cli/v3"
)

func (c *CLI) substitutionCommand() *cli.Command {
	return &cli.Command{Name: "substitution", Usage: "Create or revise an explicit ID-based substitution rule", Flags: []cli.Flag{&cli.StringFlag{Name: "id", Required: true}, &cli.StringFlag{Name: "substitute-id", Required: true}, &cli.Float64Flag{Name: "ratio", Value: 1}, &cli.StringFlag{Name: "quality", Value: "similar"}, &cli.Uint64Flag{Name: "revision"}, &cli.BoolFlag{Name: "disabled"}, &cli.StringFlag{Name: "notes"}}, Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
		id, err := entity.ParseIngredientID(cmd.String("id"))
		if err != nil {
			return err
		}
		substitute, err := entity.ParseIngredientID(cmd.String("substitute-id"))
		if err != nil {
			return err
		}
		rule, err := c.app.Ingredients.SetSubstitution(ctx, &models.SubstitutionRule{IngredientID: id, SubstituteID: substitute, Ratio: cmd.Float64("ratio"), QualityImpact: models.Quality(cmd.String("quality")), Revision: cmd.Uint64("revision"), Disabled: cmd.Bool("disabled"), Notes: cmd.String("notes")})
		if err != nil {
			return err
		}
		return toolkit.WriteJSON(cmd.Writer, rule)
	})}
}

func (c *CLI) substitutionsCommand() *cli.Command {
	return &cli.Command{Name: "substitutions", Usage: "List enabled and disabled substitution rules with revisions", Flags: []cli.Flag{&cli.StringFlag{Name: "id", Required: true}}, Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
		id, err := entity.ParseIngredientID(cmd.String("id"))
		if err != nil {
			return err
		}
		rules, err := c.app.Ingredients.SubstitutionRules(ctx, id)
		if err != nil {
			return err
		}
		return toolkit.WriteJSON(cmd.Writer, rules)
	})}
}
