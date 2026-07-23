package models

import "github.com/TheFellow/go-modular-monolith/pkg/filter"

type ListFilterView struct {
	ID          string `expr:"id" filter:"Ingredient ID" filter-column:"ID"`
	Name        string `expr:"name" filter:"Ingredient name" filter-column:"Name"`
	Category    string `expr:"category" filter:"Ingredient category" filter-column:"Category"`
	Unit        string `expr:"unit" filter:"Measurement unit" filter-column:"Unit"`
	Description string `expr:"description" filter:"Ingredient description" filter-column:"Description"`
}

func ListFilterSchema() filter.Schema[ListFilterView] {
	return filter.NewSchema[ListFilterView](
		`category == "spirit" && name.contains("gin")`,
		`unit in ["ml", "oz"] && !description.contains("seasonal")`,
	)
}
