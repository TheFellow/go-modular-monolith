package dao

type DrinkRow struct {
	ID          string
	Name        string `bstore:"unique"`
	Category    string `bstore:"index"`
	Glass       string `bstore:"index"`
	Recipe      RecipeRow
	Description string
}

type RecipeRow struct {
	Ingredients []RecipeIngredientRow
	Steps       []string
	Garnish     string
}

type RecipeIngredientRow struct {
	IngredientID string
	Amount       float64
	Unit         string
	Optional     bool
	Substitutes  []string
}
