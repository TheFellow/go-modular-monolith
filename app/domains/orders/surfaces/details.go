package surfaces

import (
	"fmt"
	"strings"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
)

// Snapshot describes the persisted evidence without looking up mutable catalog
// records. Quantities in ingredient selections already include line quantities.
func Snapshot(items []models.ItemSnapshot) string {
	var lines []string
	for _, item := range items {
		price := "N/A"
		if value, ok := item.Price.Unwrap(); ok {
			price = value.String()
		}
		lines = append(lines, fmt.Sprintf("%s · %s · quantity %d · unit price %s", item.Name, item.DrinkID, item.Quantity, price))
		if item.Notes != "" {
			lines = append(lines, "Notes: "+item.Notes)
		}
		for _, selection := range item.Ingredients {
			if selection.Omitted {
				lines = append(lines, "  Omitted optional ingredient: "+selection.OriginalID.String())
				continue
			}
			optional := ""
			if selection.Optional {
				optional = " · optional"
			}
			lines = append(lines, fmt.Sprintf("  %g %s %s · %s%s", selection.Quantity, selection.Unit, selection.Name, selection.IngredientID, optional))
			lines = append(lines, fmt.Sprintf("    Original: %s · ratio %g", selection.OriginalID, selection.Ratio))
		}
		for i, step := range item.Steps {
			lines = append(lines, fmt.Sprintf("  %d. %s", i+1, step))
		}
		lines = append(lines, "Garnish: "+item.Garnish)
	}
	return strings.Join(lines, "\n")
}
func History(records []models.AmendmentRecord) string {
	if len(records) == 0 {
		return "No amendments"
	}
	var lines []string
	for _, record := range records {
		lines = append(lines, "Amended "+record.At.Format(time.RFC3339)+" by "+record.Principal+": "+record.Reason,
			"Before", Snapshot(record.Before), "After", Snapshot(record.After))
	}
	return strings.Join(lines, "\n")
}
func Usage(usages []models.IngredientUsage) string {
	var lines []string
	for _, usage := range usages {
		lines = append(lines, fmt.Sprintf("%s · %s · %s", usage.Name, usage.IngredientID, usage.Amount))
	}
	return strings.Join(lines, "\n")
}
