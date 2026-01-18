package main

import (
	"encoding/json"
	"io"
	"strings"
	"unicode"

	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
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
		return money.NewPrice(strings.TrimPrefix(s, "$"), "USD")
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

	return money.NewPrice(number, currency)
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

func parseDrinkID(value string) (entity.DrinkID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return entity.DrinkID{}, errors.Invalidf("drink id is required")
	}
	id, err := entity.ParseDrinkID(value)
	if err != nil {
		return entity.DrinkID{}, err
	}
	return id, nil
}

func parseIngredientID(value string) (entity.IngredientID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return entity.IngredientID{}, errors.Invalidf("ingredient id is required")
	}
	id, err := entity.ParseIngredientID(value)
	if err != nil {
		return entity.IngredientID{}, err
	}
	return id, nil
}

func parseMenuID(value string) (entity.MenuID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return entity.MenuID{}, errors.Invalidf("menu id is required")
	}
	id, err := entity.ParseMenuID(value)
	if err != nil {
		return entity.MenuID{}, err
	}
	return id, nil
}

func parseOrderID(value string) (entity.OrderID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return entity.OrderID{}, errors.Invalidf("order id is required")
	}
	id, err := entity.ParseOrderID(value)
	if err != nil {
		return entity.OrderID{}, err
	}
	return id, nil
}
