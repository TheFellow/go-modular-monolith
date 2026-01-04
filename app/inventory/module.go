package inventory

import (
	ingredientsqueries "github.com/TheFellow/go-modular-monolith/app/ingredients/queries"
	"github.com/TheFellow/go-modular-monolith/app/inventory/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/inventory/queries"
)

type Module struct {
	ingredientsQueries *ingredientsqueries.Queries
	queries            *queries.Queries
	adjust             *commands.Adjust
	set                *commands.Set
}

func NewModule() *Module {
	return &Module{
		ingredientsQueries: ingredientsqueries.New(),
		queries:            queries.New(),
		adjust:             commands.NewAdjust(),
		set:                commands.NewSet(),
	}
}
