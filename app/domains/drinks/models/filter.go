package models

import "github.com/TheFellow/go-modular-monolith/pkg/filter"

type RecipeFilterView struct {
	Garnish string `expr:"garnish" filter:"Recipe garnish"`
}

type ListFilterView struct {
	ID          string           `expr:"id" filter:"Drink ID" filter-column:"ID"`
	Name        string           `expr:"name" filter:"Drink name" filter-column:"Name"`
	Category    string           `expr:"category" filter:"Drink category" filter-column:"Category"`
	Glass       string           `expr:"glass" filter:"Glass type" filter-column:"Glass"`
	Description string           `expr:"description" filter:"Drink description" filter-column:"Description"`
	Recipe      RecipeFilterView `expr:"recipe"`
}

func ListFilterSchema() filter.Schema[ListFilterView] {
	return filter.NewSchema[ListFilterView](
		`category == "cocktail" && name.contains("gin")`,
		`glass in ["coupe", "rocks"] || recipe.garnish.startsWith("lemon")`,
	)
}
