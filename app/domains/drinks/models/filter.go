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
	Status      string           `expr:"status" filter:"Lifecycle status" filter-column:"Status"`
	Description string           `expr:"description" filter:"Drink description" filter-column:"Description"`
	Tags        []string         `expr:"tags" filter:"Tags (key or key=value)"`
	Recipe      RecipeFilterView `expr:"recipe"`
}

func ListFilterSchema() filter.Schema[ListFilterView] {
	return filter.NewSchema[ListFilterView](
		`category == "cocktail" && name.contains("gin")`,
		`glass in ["coupe", "rocks"] || recipe.garnish.startsWith("lemon")`,
		`status == "review_required"`,
		`tags contains "featured" || tags contains "region=west"`,
	)
}
