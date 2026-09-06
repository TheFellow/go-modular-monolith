package models

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"time"
)

type Disposal struct {
	IngredientID entity.IngredientID
	Revision     uint64
	Amount       measurement.Amount
	Reason       string
}

// Movement keeps physical quantity and disposition evidence independently of
// whether the ingredient is still offered in the active catalog.
type Movement struct {
	ID           string
	InventoryID  entity.InventoryID
	At           time.Time
	Before       float64
	After        float64
	Unit         measurement.Unit
	BeforeStatus Status
	AfterStatus  Status
	Reason       string
}

// Disposition either quarantines retained stock or releases its quarantine.
type Disposition struct {
	IngredientID entity.IngredientID
	Revision     uint64
	Quarantine   bool
	Reason       string
}
