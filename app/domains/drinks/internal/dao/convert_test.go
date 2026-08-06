package dao

import (
	"context"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
	"github.com/stretchr/testify/require"
)

func TestToModelRejectsInvalidPersistedStatusAsInternal(t *testing.T) {
	t.Parallel()
	_, err := toModel(DrinkRow{ID: "corrupt", Status: "surprising"})
	require.Error(t, err)
	require.True(t, apperrors.IsInternal(err))
	require.Contains(t, err.Error(), "invalid persisted status")
}

func TestRegisterExplicitlyBackfillsLegacyStatus(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s, err := store.Open(ctx, t.TempDir()+"/legacy.db")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })

	s.Register(ctx, DrinkRow{})
	require.NoError(t, s.Write(ctx, func(tx *bstore.Tx) error {
		return tx.Insert(&DrinkRow{ID: "legacy", Name: "Legacy"})
	}))

	require.NoError(t, backfillLegacyStatuses(ctx, s))
	require.NoError(t, s.Read(ctx, func(tx *bstore.Tx) error {
		row := DrinkRow{ID: "legacy"}
		require.NoError(t, tx.Get(&row))
		require.Equal(t, string(models.StatusActive), row.Status)
		return nil
	}))
}
