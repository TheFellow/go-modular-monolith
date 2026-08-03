package main

import (
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	appfilter "github.com/TheFellow/go-modular-monolith/pkg/filter"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/urfave/cli/v3"
)

func listPagingFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{Name: "limit", Usage: "Number of entries in a cursor page (default 100)"},
		&cli.StringFlag{Name: "cursor", Usage: "Continue after a result cursor"},
	}
}

func filterFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "filter", Usage: "Filter expression (run with --filter-help for fields and examples)"},
		&cli.BoolFlag{Name: "filter-help", Usage: "Show filter fields, syntax, and examples, then exit"},
	}
}

func appendFilterFlags(flags []cli.Flag) []cli.Flag {
	return append(flags, filterFlags()...)
}

func tagsFlag() cli.Flag {
	return &cli.StringFlag{
		Name:  "tags",
		Usage: "Complete tag set as CSV (for example: region=east,env=dev,terraform)",
	}
}

func appendTagsFlag(flags []cli.Flag) []cli.Flag {
	return append(flags, tagsFlag())
}

// runTaggedMutation keeps a domain mutation and its optional complete tag-set
// replacement in one caller-owned transaction. Parsing happens first so bad
// input cannot execute the domain mutation. An omitted flag preserves tags;
// an explicitly empty flag replaces them with the empty set.
func runTaggedMutation[T app.TaggableEntity](
	c *CLI,
	ctx *middleware.Context,
	cmd *cli.Command,
	mutate func(*middleware.Context) (T, error),
) (T, error) {
	var zero T
	if !cmd.IsSet("tags") {
		return mutate(ctx)
	}

	desired, err := tag.ParseCollection(cmd.String("tags"))
	if err != nil {
		return zero, err
	}

	return app.RunTaggedMutation(c.app, ctx, &desired, mutate)
}

func filterAction[T any](c *CLI, schema appfilter.Schema[T], fn func(*middleware.Context, *cli.Command) error) cli.ActionFunc {
	return c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
		if cmd.Bool("filter-help") {
			return writeFilterHelp(cmd.Writer, schema)
		}
		return fn(ctx, cmd)
	})
}

func writeFilterHelp[T any](w io.Writer, schema appfilter.Schema[T]) error {
	if _, err := fmt.Fprintln(w, "FILTER SYNTAX"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "  Comparisons: ==  !=  <  <=  >  >=  in  not in"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "  Logic:       && / and   || / or   ! / not   (parentheses)"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "  Strings:     value.contains(\"x\"), startsWith, endsWith, matches"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "               Infix aliases are also accepted: value contains \"x\""); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "  Time:        date(\"2026-07-01T00:00:00Z\")"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "\nFIELDS"); err != nil {
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, field := range schema.Fields() {
		if _, err := fmt.Fprintf(tw, "  %s\t%s\t%s\n", field.Name, filterTypeName(field.Type), field.Description); err != nil {
			return err
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	examples := schema.Examples()
	if len(examples) > 0 {
		if _, err := fmt.Fprintln(w, "\nEXAMPLES"); err != nil {
			return err
		}
		for _, example := range examples {
			if _, err := fmt.Fprintf(w, "  --filter '%s'\n", example); err != nil {
				return err
			}
		}
	}
	return nil
}

func filterTypeName(t reflect.Type) string {
	if t == reflect.TypeFor[time.Time]() {
		return "timestamp"
	}
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		return "list<" + filterTypeName(t.Elem()) + ">"
	}
	if t.Kind() == reflect.Pointer {
		return filterTypeName(t.Elem()) + "?"
	}
	if t.Name() != "" && t.PkgPath() == "" {
		return t.Name()
	}
	return t.Kind().String()
}

func pagingRequest(cmd *cli.Command) paging.Request {
	return paging.Request{
		Cursor: paging.Cursor(strings.TrimSpace(cmd.String("cursor"))),
		Limit:  cmd.Int("limit"),
	}
}

func requiredStringArg(cmd *cli.Command, name string) (string, error) {
	values := cmd.StringArgs(name)
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		return "", cli.Exit(fmt.Sprintf("%s argument is required", name), errors.ExitUsage)
	}
	return values[0], nil
}

func printNextCursor(w io.Writer, cursor paging.Cursor) error {
	if cursor == "" {
		return nil
	}
	_, err := fmt.Fprintf(w, "Next cursor: %s\n", cursor)
	return err
}

var (
	CostsFlag        cli.Flag = &cli.BoolFlag{Name: "costs", Usage: "Include cost/margin analytics"}
	TargetMarginFlag cli.Flag = &cli.Float64Flag{
		Name: "target-margin", Usage: "Target margin for suggested prices (0-1)", Value: 0.7,
		Validator: func(value float64) error {
			if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 || value >= 1 {
				return errors.Invalidf("target margin must be a number between 0 and 1")
			}
			return nil
		},
	}
)

func newTabWriter(output io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)
}

func parsePrice(s string) (money.Price, error) {
	return money.ParsePrice(s)
}
