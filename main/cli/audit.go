package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	auditmodels "github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/urfave/cli/v3"
)

func (c *CLI) auditCommands() *cli.Command {
	return &cli.Command{
		Name:  "audit",
		Usage: "Audit log",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List audit entries",
				Flags: auditListFlags(),
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					req, err := auditListRequest(cmd)
					if err != nil {
						return err
					}
					entries, err := c.app.Audit.List(ctx, req)
					if err != nil {
						return err
					}
					return printAuditEntries(entries)
				}),
			},
			{
				Name:      "history",
				Usage:     "List audit entries for an entity",
				Arguments: []cli.Argument{&cli.StringArgs{Name: "entity", UsageText: "Entity UID (Type::id)", Min: 1, Max: 1}},
				Flags:     auditHistoryFlags(),
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					entityArg := cmd.StringArgs("entity")[0]
					entityID, err := parseEntityUID(entityArg)
					if err != nil {
						return err
					}
					req, err := auditListRequest(cmd)
					if err != nil {
						return err
					}
					req.Entity = entityID
					entries, err := c.app.Audit.List(ctx, req)
					if err != nil {
						return err
					}
					return printAuditEntries(entries)
				}),
			},
			{
				Name:      "actor",
				Usage:     "List audit entries for an actor",
				Arguments: []cli.Argument{&cli.StringArgs{Name: "actor", UsageText: "Actor (owner|anonymous) or Entity UID", Min: 1, Max: 1}},
				Flags:     auditHistoryFlags(),
				Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
					actorArg := cmd.StringArgs("actor")[0]
					principal, err := parsePrincipal(actorArg)
					if err != nil {
						return err
					}
					req, err := auditListRequest(cmd)
					if err != nil {
						return err
					}
					req.Principal = principal
					entries, err := c.app.Audit.List(ctx, req)
					if err != nil {
						return err
					}
					return printAuditEntries(entries)
				}),
			},
		},
	}
}

func auditListFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "entity",
			Usage: "Filter by entity (Type::id)",
		},
		&cli.StringFlag{
			Name:  "principal",
			Usage: "Filter by principal (owner|anonymous or Type::id)",
		},
		&cli.StringFlag{
			Name:  "action",
			Usage: "Filter by action (Type::Action::id)",
		},
		&cli.StringFlag{
			Name:  "from",
			Usage: "Filter by start time (RFC3339 or YYYY-MM-DD)",
		},
		&cli.StringFlag{
			Name:  "to",
			Usage: "Filter by end time (RFC3339 or YYYY-MM-DD)",
		},
		&cli.IntFlag{
			Name:  "limit",
			Usage: "Limit number of entries",
		},
	}
}

func auditHistoryFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "Filter by start time (RFC3339 or YYYY-MM-DD)",
		},
		&cli.StringFlag{
			Name:  "to",
			Usage: "Filter by end time (RFC3339 or YYYY-MM-DD)",
		},
		&cli.IntFlag{
			Name:  "limit",
			Usage: "Limit number of entries",
		},
	}
}

func auditListRequest(cmd *cli.Command) (audit.ListRequest, error) {
	var req audit.ListRequest
	req.Limit = cmd.Int("limit")

	if raw := strings.TrimSpace(cmd.String("entity")); raw != "" {
		uid, err := parseEntityUID(raw)
		if err != nil {
			return req, err
		}
		req.Entity = uid
	}
	if raw := strings.TrimSpace(cmd.String("principal")); raw != "" {
		uid, err := parsePrincipal(raw)
		if err != nil {
			return req, err
		}
		req.Principal = uid
	}
	if raw := strings.TrimSpace(cmd.String("action")); raw != "" {
		uid, err := parseEntityUID(raw)
		if err != nil {
			return req, err
		}
		req.Action = uid
	}
	if raw := strings.TrimSpace(cmd.String("from")); raw != "" {
		t, err := parseTimeFilter(raw)
		if err != nil {
			return req, err
		}
		req.From = t
	}
	if raw := strings.TrimSpace(cmd.String("to")); raw != "" {
		t, err := parseTimeFilter(raw)
		if err != nil {
			return req, err
		}
		req.To = t
	}
	return req, nil
}

func parseTimeFilter(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid time %q", value)
}

func parsePrincipal(value string) (cedar.EntityUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return cedar.EntityUID{}, nil
	}
	if !strings.Contains(value, "::") {
		if uid, err := authn.ParseActor(value); err == nil {
			return uid, nil
		}
	}
	return parseEntityUID(value)
}

func parseEntityUID(value string) (cedar.EntityUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return cedar.EntityUID{}, nil
	}
	if strings.Contains(value, "::\"") || strings.HasSuffix(value, "\"") {
		var uid cedar.EntityUID
		if err := uid.UnmarshalCedar([]byte(value)); err != nil {
			return cedar.EntityUID{}, err
		}
		return uid, nil
	}
	idx := strings.LastIndex(value, "::")
	if idx <= 0 || idx+2 >= len(value) {
		return cedar.EntityUID{}, fmt.Errorf("invalid entity uid %q", value)
	}
	typ := value[:idx]
	id := strings.Trim(value[idx+2:], "\"")
	if typ == "" || id == "" {
		return cedar.EntityUID{}, fmt.Errorf("invalid entity uid %q", value)
	}
	return cedar.NewEntityUID(cedar.EntityType(typ), cedar.String(id)), nil
}

func printAuditEntries(entries []*auditmodels.AuditEntry) error {
	w := newTabWriter()
	fmt.Fprintln(w, "ID\tSTARTED_AT\tACTION\tRESOURCE\tPRINCIPAL\tSUCCESS\tTOUCHES\tERROR")
	for _, entry := range entries {
		errText := entry.Error
		fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\t%t\t%d\t%s\n",
			entry.ID.String(),
			entry.StartedAt.Format(time.RFC3339),
			entry.Action,
			entry.Resource.String(),
			entry.Principal.String(),
			entry.Success,
			len(entry.Touches),
			errText,
		)
	}
	return w.Flush()
}
