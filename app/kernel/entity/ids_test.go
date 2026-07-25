package entity_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestParseIDRejectsEmptyValue(t *testing.T) {
	t.Parallel()

	_, err := entity.ParseAuditEntryID("")
	testutil.ErrorIsInvalid(t, err)
}

func TestParseIDInfersEveryGeneratedEntityType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		uid  cedar.EntityUID
	}{
		{"drink", entity.NewDrinkID().EntityUID()},
		{"ingredient", entity.NewIngredientID().EntityUID()},
		{"menu", entity.NewMenuID().EntityUID()},
		{"order", entity.NewOrderID().EntityUID()},
		{"inventory", entity.NewInventoryID().EntityUID()},
		{"audit entry", entity.NewAuditEntryID().EntityUID()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := entity.ParseID(string(tt.uid.ID))
			testutil.Ok(t, err)
			testutil.Equals(t, got, tt.uid)
		})
	}
}

func TestParseIDRejectsInvalidAndUnknownIDs(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "missing-separator", "wat-3BxsD9vQRgeYqJ8v4bFVvytN", "drk-not-a-ksuid"} {
		_, err := entity.ParseID(value)
		testutil.ErrorIsInvalid(t, err)
	}
}
