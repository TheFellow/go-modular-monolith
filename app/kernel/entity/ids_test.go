package entity_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestParseIDRejectsEmptyValue(t *testing.T) {
	t.Parallel()

	_, err := entity.ParseAuditEntryID("")
	testutil.ErrorIsInvalid(t, err)
}
