package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
	"time"
	"unicode"

	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
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
	JSONFlag         cli.Flag = &cli.BoolFlag{Name: "json", Usage: "Output JSON"}
	TemplateFlag     cli.Flag = &cli.BoolFlag{Name: "template", Usage: "Print JSON template and exit"}
	StdinFlag        cli.Flag = &cli.BoolFlag{Name: "stdin", Usage: "Read JSON from stdin"}
	FileFlag         cli.Flag = &cli.StringFlag{Name: "file", Usage: "Read JSON from file"}
	CostsFlag        cli.Flag = &cli.BoolFlag{Name: "costs", Usage: "Include cost/margin analytics"}
	TargetMarginFlag cli.Flag = &cli.Float64Flag{Name: "target-margin", Usage: "Target margin for suggested prices (0-1)", Value: 0.7}
)

func writeJSON(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = w.Write(b)
	return err
}

func newTabWriter(output io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)
}

func readJSONInput[T any](cmd *cli.Command) (T, error) {
	var zero T

	fromStdin := cmd.Bool("stdin")
	fromFile := strings.TrimSpace(cmd.String("file"))
	if fromStdin && fromFile != "" {
		return zero, fmt.Errorf("set only one of --stdin or --file")
	}
	if !fromStdin && fromFile == "" {
		return zero, fmt.Errorf("missing input: set --stdin or --file (or use --template)")
	}

	var r io.Reader
	if fromStdin {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return zero, err
		}
		if len(bytes.TrimSpace(b)) == 0 {
			return zero, fmt.Errorf("stdin is empty")
		}
		r = bytes.NewReader(b)
	} else {
		f, err := os.Open(fromFile)
		if err != nil {
			return zero, err
		}
		defer func() { _ = f.Close() }()
		r = f
	}

	var result T
	if err := json.NewDecoder(r).Decode(&result); err != nil {
		return zero, err
	}
	return result, nil
}

func parsePrice(s string) (money.Price, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return money.Price{}, errors.Invalidf("price is required")
	}

	if after, ok := strings.CutPrefix(s, "$"); ok {
		return money.NewPrice(after, currency.USD)
	}

	parts := strings.Fields(s)
	if len(parts) != 2 {
		return money.Price{}, errors.Invalidf("invalid price %q (expected \"$1.23\" or \"USD 1.23\" or \"1.23 USD\")", s)
	}

	var currencyCode, number string
	if isCurrency(parts[0]) {
		currencyCode, number = parts[0], parts[1]
	} else if isCurrency(parts[1]) {
		currencyCode, number = parts[1], parts[0]
	} else {
		return money.Price{}, errors.Invalidf("invalid price %q (expected \"$1.23\" or \"USD 1.23\" or \"1.23 USD\")", s)
	}

	curr, err := currency.Parse(currencyCode)
	if err != nil {
		return money.Price{}, err
	}
	return money.NewPrice(number, curr)
}

func isCurrency(s string) bool {
	if len(s) != 3 {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}
