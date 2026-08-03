package models

import (
	inventoryauthz "github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	cedar "github.com/cedar-policy/cedar-go"
)

type Update struct {
	IngredientID entity.IngredientID
	Amount       measurement.Amount
	CostPerUnit  money.Price
}

func (u Update) EntityUID() cedar.EntityUID {
	return cedar.NewEntityUID(InventoryEntityType, cedar.String(""))
}

func (u Update) CedarEntity() cedar.Entity {
	return inventoryauthz.Inventory{
		UID:          u.EntityUID(),
		IngredientID: u.IngredientID.EntityUID(),
		Unit:         string(u.Amount.Unit()),
	}.CedarEntity()
}
