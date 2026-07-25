package main

import (
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
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
						return writeJSON(cmd.Writer, out)
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
		return writeJSON(cmd.Writer, out)
	}
	state := "unchanged"
	if changed {
		state = "changed"
	}
	return printTagState(cmd, out, state)
}

func printTagState(cmd *cli.Command, out tagsOutput, state string) error {
	values := out.Tags.String()
	if values == "" {
		values = "(none)"
	}
	if state == "" {
		_, err := fmt.Fprintf(cmd.Writer, "%s: %s\n", out.EntityID, values)
		return err
	}
	_, err := fmt.Fprintf(cmd.Writer, "%s: %s (%s)\n", out.EntityID, values, state)
	return err
}
