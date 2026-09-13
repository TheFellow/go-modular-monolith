// Package surfaces contains framework-independent order presentation data.
package surfaces

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

type ReplacementField struct {
	ID                         entity.IngredientID
	Name, ReplacementID, Ratio string
}
type PreparationField struct {
	ID                   entity.DrinkID
	Name, Steps, Garnish string
	initialSteps         string
	initialGarnish       string
}

// AmendmentForm retains the revision shown when the editor was opened.
// Blank replacement IDs preserve that ingredient; preparation starts from the
// current approved values, so clearing a garnish is an explicit amendment.
type AmendmentForm struct {
	OrderID      entity.OrderID
	Revision     uint64
	Reason       string
	Replacements []ReplacementField
	Preparation  []PreparationField
}

func NewAmendmentForm(order models.Order) AmendmentForm {
	f := AmendmentForm{OrderID: order.ID, Revision: order.Revision}
	seen := map[entity.IngredientID]bool{}
	drinks := map[entity.DrinkID]bool{}
	for _, item := range order.Plan {
		for _, ingredient := range item.Ingredients {
			if ingredient.Omitted || seen[ingredient.IngredientID] {
				continue
			}
			seen[ingredient.IngredientID] = true
			f.Replacements = append(f.Replacements, ReplacementField{ID: ingredient.IngredientID, Name: ingredient.Name, Ratio: "1"})
		}
		if !drinks[item.DrinkID] {
			drinks[item.DrinkID] = true
			steps := strings.Join(item.Steps, "\n")
			f.Preparation = append(f.Preparation, PreparationField{ID: item.DrinkID, Name: item.Name, Steps: steps, Garnish: item.Garnish, initialSteps: steps, initialGarnish: item.Garnish})
		}
	}
	return f
}
func (f AmendmentForm) Clone() AmendmentForm {
	f.Replacements = slices.Clone(f.Replacements)
	f.Preparation = slices.Clone(f.Preparation)
	return f
}
func (f AmendmentForm) Request() (models.Amendment, error) {
	request := models.Amendment{OrderID: f.OrderID, Revision: f.Revision, Reason: strings.TrimSpace(f.Reason)}
	if request.Reason == "" {
		return request, errors.Invalidf("amendment reason is required")
	}
	for _, field := range f.Replacements {
		if strings.TrimSpace(field.ReplacementID) == "" {
			continue
		}
		id, err := entity.ParseIngredientID(strings.TrimSpace(field.ReplacementID))
		if err != nil {
			return request, errors.Invalidf("replacement for %s: %w", field.Name, err)
		}
		ratio, err := strconv.ParseFloat(strings.TrimSpace(field.Ratio), 64)
		if err != nil || ratio <= 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
			return request, errors.Invalidf("replacement ratio for %s must be finite and positive", field.Name)
		}
		if id == field.ID {
			return request, errors.Invalidf("replacement for %s must be a different ingredient", field.Name)
		}
		request.Replacements = append(request.Replacements, models.Replacement{OriginalID: field.ID, ReplacementID: id, Ratio: ratio})
	}
	if len(request.Replacements) == 0 {
		return request, errors.Invalidf("at least one ingredient replacement is required")
	}
	for _, field := range f.Preparation {
		preparation := models.PreparationAmendment{DrinkID: field.ID}
		if field.Steps != field.initialSteps {
			steps := strings.Split(strings.TrimSpace(field.Steps), "\n")
			for i, step := range steps {
				steps[i] = strings.TrimSpace(step)
				if steps[i] == "" {
					return request, errors.Invalidf("preparation steps for %s cannot be blank", field.Name)
				}
			}
			preparation.Steps = steps
		}
		if field.Garnish != field.initialGarnish {
			preparation.Garnish = optional.Some(field.Garnish)
		}
		if preparation.Steps != nil || preparation.Garnish.IsSome() {
			request.Preparation = append(request.Preparation, preparation)
		}
	}
	return request, nil
}

// Queue replaces an existing draft for the same order, avoiding duplicate
// targets while preserving the exact revision captured by each editor.
func Queue(queue []models.Amendment, request models.Amendment) []models.Amendment {
	for i := range queue {
		if queue[i].OrderID == request.OrderID {
			queue[i] = request
			return queue
		}
	}
	return append(queue, request)
}
func BatchSummary(queue []models.Amendment) string {
	lines := make([]string, 0, len(queue))
	for _, request := range queue {
		lines = append(lines, fmt.Sprintf("%s · revision %d · %s", request.OrderID, request.Revision, request.Reason))
		for _, replacement := range request.Replacements {
			lines = append(lines, fmt.Sprintf("  Replace %s with %s · ratio %g", replacement.OriginalID, replacement.ReplacementID, replacement.Ratio))
		}
		for _, preparation := range request.Preparation {
			lines = append(lines, "  Preparation: "+preparation.DrinkID.String())
			for i, step := range preparation.Steps {
				lines = append(lines, fmt.Sprintf("    %d. %s", i+1, step))
			}
			if garnish, ok := preparation.Garnish.Unwrap(); ok {
				lines = append(lines, "    Garnish: "+garnish)
			}
		}
	}
	return strings.Join(lines, "\n")
}
