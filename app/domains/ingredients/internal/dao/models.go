package dao

import "time"

type IngredientRow struct {
	ID          string
	Revision    uint64 `json:"-" store:"revision"`
	Name        string `store:"unique"`
	Category    string `store:"index"`
	Unit        string
	Description string
	DeletedAt   *time.Time
}
