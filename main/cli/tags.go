package main

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	clitoolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/cli"
	clitable "github.com/TheFellow/go-modular-monolith/pkg/toolkits/cli/table"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/urfave/cli/v3"
)

type tagsOutput struct {
	EntityID string               `json:"entity_id"`
	Tags     tag.CanonicalStrings `json:"tags"`
	Changed  *bool                `json:"changed,omitempty"`
}

func (c *CLI) tagsCommands() *cli.Command {
	return &cli.Command{
		Name:  "tags",
		Usage: "Manage entity tags",
		Commands: []*cli.Command{
			{
				Name:      "show",
				Usage:     "Show active entities referencing a tag",
				UsageText: "mixology tags show [--json] <key[=value]>\n   or: mixology tags show [--json] --key <key>",
				Arguments: []cli.Argument{&cli.StringArg{Name: "tag", UsageText: "<key[=value]>"}},
				Flags: []cli.Flag{
					clitoolkit.JSONFlag,
					&cli.StringFlag{Name: "key", Usage: "Match every value for this tag key"},
				},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					rawTag := strings.TrimSpace(cmd.StringArg("tag"))
					rawKey := strings.TrimSpace(cmd.String("key"))
					if rawTag == "" && rawKey == "" {
						return cli.Exit("tag argument or --key is required", errors.ExitUsage)
					}
					if rawTag != "" && rawKey != "" {
						return cli.Exit("tag argument and --key cannot be used together", errors.ExitUsage)
					}
					exact := rawTag != ""
					var value tag.Tag
					var err error
					if exact {
						value, err = tag.Parse(rawTag)
					} else {
						value, err = tag.New(rawKey, "")
					}
					if err != nil {
						return err
					}
					rows, err := c.app.Tags.Show(ctx, value, exact)
					if err != nil {
						return err
					}
					if cmd.Bool("json") {
						return clitoolkit.WriteJSON(cmd.Writer, rows)
					}
					return clitable.PrintTable(cmd.Writer, rows)
				}),
			},
			{
				Name:  "summary",
				Usage: "Summarize active tag usage",
				Flags: []cli.Flag{clitoolkit.JSONFlag},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					rows, err := c.app.Tags.Summary(ctx)
					if err != nil {
						return err
					}
					if cmd.Bool("json") {
						return clitoolkit.WriteJSON(cmd.Writer, rows)
					}
					return clitable.PrintTable(cmd.Writer, rows)
				}),
			},
			{
				Name:      "add",
				Usage:     "Add or replace a tag",
				UsageText: "mixology tags add [--json] <entity-id> <key[=value]>",
				Arguments: []cli.Argument{
					&cli.StringArg{Name: "entity-id", UsageText: "<entity-id>"},
					&cli.StringArg{Name: "tag", UsageText: "<key[=value]>"},
				},
				Flags: []cli.Flag{&cli.BoolFlag{Name: "json", Usage: "Output JSON"}},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					target, err := tagTargetArg(cmd)
					if err != nil {
						return err
					}
					raw, err := requiredTagArg(cmd, "tag")
					if err != nil {
						return err
					}
					value, err := tag.Parse(raw)
					if err != nil {
						return err
					}
					result, err := c.app.Tags.Upsert(ctx, target, value)
					if err != nil {
						return err
					}
					return printTagMutation(cmd, result)
				}),
			},
			{
				Name:      "remove",
				Usage:     "Remove a tag by key",
				UsageText: "mixology tags remove [--json] <entity-id> <key>",
				Arguments: []cli.Argument{
					&cli.StringArg{Name: "entity-id", UsageText: "<entity-id>"},
					&cli.StringArg{Name: "key", UsageText: "<key>"},
				},
				Flags: []cli.Flag{&cli.BoolFlag{Name: "json", Usage: "Output JSON"}},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					target, err := tagTargetArg(cmd)
					if err != nil {
						return err
					}
					key, err := requiredTagArg(cmd, "key")
					if err != nil {
						return err
					}
					result, err := c.app.Tags.Remove(ctx, target, key)
					if err != nil {
						return err
					}
					return printTagMutation(cmd, result)
				}),
			},
			{
				Name:      "list",
				Usage:     "List tags on an entity",
				UsageText: "mixology tags list [--json] <entity-id>",
				Arguments: []cli.Argument{
					&cli.StringArg{Name: "entity-id", UsageText: "<entity-id>"},
				},
				Flags: []cli.Flag{&cli.BoolFlag{Name: "json", Usage: "Output JSON"}},
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					target, err := tagTargetArg(cmd)
					if err != nil {
						return err
					}
					values, err := c.app.Tags.List(ctx, target)
					if err != nil {
						return err
					}
					out := tagsOutput{EntityID: string(target.ID), Tags: values.Canonical()}
					if cmd.Bool("json") {
						return clitoolkit.WriteJSON(cmd.Writer, out)
					}
					return printTagState(cmd, out, "")
				}),
			},
		},
	}
}

func tagTargetArg(cmd *cli.Command) (cedar.EntityUID, error) {
	raw, err := requiredTagArg(cmd, "entity-id")
	if err != nil {
		return cedar.EntityUID{}, err
	}
	return entity.ParseID(raw)
}

func requiredTagArg(cmd *cli.Command, name string) (string, error) {
	value := strings.TrimSpace(cmd.StringArg(name))
	if value == "" {
		return "", cli.Exit(fmt.Sprintf("%s argument is required", name), errors.ExitUsage)
	}
	return value, nil
}

func printTagMutation(cmd *cli.Command, result tagging.Result) error {
	changed := result.Changed
	out := tagsOutput{
		EntityID: string(result.Target.ID),
		Tags:     result.Tags.Canonical(),
		Changed:  &changed,
	}
	if cmd.Bool("json") {
		return clitoolkit.WriteJSON(cmd.Writer, out)
	}
	state := "unchanged"
	if changed {
		state = "changed"
	}
	return printTagState(cmd, out, state)
}

func printTagState(cmd *cli.Command, out tagsOutput, state string) error {
	values := cmp.Or(out.Tags.String(), "(none)")
	if state == "" {
		_, err := fmt.Fprintf(cmd.Writer, "%s: %s\n", out.EntityID, values)
		return err
	}
	_, err := fmt.Fprintf(cmd.Writer, "%s: %s (%s)\n", out.EntityID, values, state)
	return err
}
