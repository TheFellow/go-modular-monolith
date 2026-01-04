package dao

type IngredientRow struct {
	ID          string
	Name        string `bstore:"unique"`
	Category    string `bstore:"index"`
	Unit        string
	Description string
}
