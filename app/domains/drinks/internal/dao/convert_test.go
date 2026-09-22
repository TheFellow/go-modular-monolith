package dao

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestToModelRejectsInvalidPersistedStatusAsInternal(t *testing.T) {
	t.Parallel()
	_, err := toModel(DrinkRow{ID: "corrupt", Status: "surprising"})
	require.Error(t, err)
	require.True(t, errors.IsInternal(err))
	require.Contains(t, err.Error(), "invalid persisted status")
}
