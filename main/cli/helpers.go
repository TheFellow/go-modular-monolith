package main

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/TheFellow/go-modular-monolith/app/money"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/urfave/cli/v3"
)

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

func parsePrice(s string) (money.Price, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return money.Price{}, errors.Invalidf("price is required")
	}

	if strings.HasPrefix(s, "$") {
		amount, err := parseDecimalToCents(strings.TrimPrefix(s, "$"))
		if err != nil {
			return money.Price{}, err
		}
		p := money.Price{Amount: amount, Currency: "USD"}
		return p, p.Validate()
	}

	parts := strings.Fields(s)
	if len(parts) != 2 {
		return money.Price{}, errors.Invalidf("invalid price %q (expected \"$1.23\" or \"USD 1.23\" or \"1.23 USD\")", s)
	}

	var currency, number string
	if isCurrency(parts[0]) {
		currency, number = parts[0], parts[1]
	} else if isCurrency(parts[1]) {
		currency, number = parts[1], parts[0]
	} else {
		return money.Price{}, errors.Invalidf("invalid price %q (expected \"$1.23\" or \"USD 1.23\" or \"1.23 USD\")", s)
	}

	amount, err := parseDecimalToCents(number)
	if err != nil {
		return money.Price{}, err
	}
	p := money.Price{Amount: amount, Currency: strings.ToUpper(currency)}
	return p, p.Validate()
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

func parseDecimalToCents(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.Invalidf("amount is required")
	}
	if strings.HasPrefix(s, "+") {
		s = strings.TrimPrefix(s, "+")
	}
	if strings.HasPrefix(s, "-") {
		return 0, errors.Invalidf("amount must be >= 0")
	}

	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return 0, errors.Invalidf("invalid amount %q", s)
	}

	wholeStr := parts[0]
	fracStr := ""
	if len(parts) == 2 {
		fracStr = parts[1]
	}

	if wholeStr == "" {
		wholeStr = "0"
	}
	whole, err := strconv.Atoi(wholeStr)
	if err != nil || whole < 0 {
		return 0, errors.Invalidf("invalid amount %q", s)
	}

	if len(fracStr) > 2 {
		return 0, errors.Invalidf("invalid amount %q (too many decimal places)", s)
	}
	for _, r := range fracStr {
		if r < '0' || r > '9' {
			return 0, errors.Invalidf("invalid amount %q", s)
		}
	}
	if len(fracStr) == 1 {
		fracStr = fracStr + "0"
	}
	if fracStr == "" {
		fracStr = "00"
	}
	frac, err := strconv.Atoi(fracStr)
	if err != nil || frac < 0 {
		return 0, errors.Invalidf("invalid amount %q", s)
	}

	return whole*100 + frac, nil
}
