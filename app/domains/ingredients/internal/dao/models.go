package dao

import "time"

type IngredientRow struct {
	ID          string
	Name        string `store:"unique"`
	Category    string `store:"index"`
	Unit        string
	Description string
	DeletedAt   *time.Time
}
