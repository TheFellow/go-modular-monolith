package tagging_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

type testContext struct {
	context.Context
	tx *bstore.Tx
}

func (c testContext) Transaction() (*bstore.Tx, bool) { return c.tx, c.tx != nil }

func TestRepositoryUpsertIsDeterministicAndIsolatesTargets(t *testing.T) {
	t.Parallel()

	repo, s, ctx := newRepository(t)
	drink := entity.NewDrinkID().EntityUID()
	otherDrink := entity.NewDrinkID().EntityUID()
	ingredient := entity.NewIngredientID().EntityUID()

	changed := upsert(t, s, ctx, repo, drink, tag.Tag{Key: "season", Value: "summer"})
	testutil.IsTrue(t, changed)
	changed = upsert(t, s, ctx, repo, drink, tag.Tag{Key: "featured"})
	testutil.IsTrue(t, changed)
	upsert(t, s, ctx, repo, otherDrink, tag.Tag{Key: "season", Value: "winter"})
	upsert(t, s, ctx, repo, ingredient, tag.Tag{Key: "season", Value: "spring"})

	got, err := repo.List(ctx, drink)
	testutil.Ok(t, err)
	testutil.Equals(t, got, tag.Tags{
		{Key: "featured"},
		{Key: "season", Value: "summer"},
	})

	changed = upsert(t, s, ctx, repo, drink, tag.Tag{Key: "season", Value: "summer"})
	testutil.IsFalse(t, changed)
	changed = upsert(t, s, ctx, repo, drink, tag.Tag{Key: "season", Value: "autumn"})
	testutil.IsTrue(t, changed)

	got, err = repo.List(ctx, drink)
	testutil.Ok(t, err)
	testutil.Equals(t, got, tag.Tags{
		{Key: "featured"},
		{Key: "season", Value: "autumn"},
	})
	got, err = repo.List(ctx, otherDrink)
	testutil.Ok(t, err)
	testutil.Equals(t, got, tag.Tags{{Key: "season", Value: "winter"}})
	got, err = repo.List(ctx, ingredient)
	testutil.Ok(t, err)
	testutil.Equals(t, got, tag.Tags{{Key: "season", Value: "spring"}})
}

func TestRepositoryListTypeTxBatchesEntityReads(t *testing.T) {
	t.Parallel()

	repo, s, ctx := newRepository(t)
	drinkA := entity.NewDrinkID().EntityUID()
	drinkB := entity.NewDrinkID().EntityUID()
	drinkWithoutTags := entity.NewDrinkID().EntityUID()
	ingredient := entity.NewIngredientID().EntityUID()
	upsert(t, s, ctx, repo, drinkA, tag.Tag{Key: "b", Value: "2"})
	upsert(t, s, ctx, repo, drinkA, tag.Tag{Key: "a", Value: "1"})
	upsert(t, s, ctx, repo, drinkB, tag.Tag{Key: "c", Value: "3"})
	upsert(t, s, ctx, repo, ingredient, tag.Tag{Key: "a", Value: "ingredient"})

	var got map[cedar.EntityUID]tag.Tags
	err := s.Read(ctx, func(tx *bstore.Tx) error {
		var err error
		got, err = repo.ListTypeTx(tx, entity.TypeDrink, []cedar.String{
			drinkB.ID, drinkWithoutTags.ID, drinkA.ID,
		})
		return err
	})
	testutil.Ok(t, err)
	testutil.Equals(t, got, map[cedar.EntityUID]tag.Tags{
		drinkA: {{Key: "a", Value: "1"}, {Key: "b", Value: "2"}},
		drinkB: {{Key: "c", Value: "3"}},
	})

	err = s.Read(ctx, func(tx *bstore.Tx) error {
		empty, err := repo.ListTypeTx(tx, entity.TypeDrink, nil)
		testutil.Equals(t, empty, map[cedar.EntityUID]tag.Tags{})
		return err
	})
	testutil.Ok(t, err)
}

func TestRepositoryRemoveAndDeleteTargetAreIdempotent(t *testing.T) {
	t.Parallel()

	repo, s, ctx := newRepository(t)
	target := entity.NewMenuID().EntityUID()
	other := entity.NewMenuID().EntityUID()
	upsert(t, s, ctx, repo, target, tag.Tag{Key: "a", Value: "1"})
	upsert(t, s, ctx, repo, target, tag.Tag{Key: "b", Value: "2"})
	upsert(t, s, ctx, repo, other, tag.Tag{Key: "a", Value: "other"})

	changed := remove(t, s, ctx, repo, target, "a")
	testutil.IsTrue(t, changed)
	changed = remove(t, s, ctx, repo, target, "a")
	testutil.IsFalse(t, changed)

	deleted := deleteTarget(t, s, ctx, repo, target)
	testutil.Equals(t, deleted, 1)
	deleted = deleteTarget(t, s, ctx, repo, target)
	testutil.Equals(t, deleted, 0)

	got, err := repo.List(ctx, target)
	testutil.Ok(t, err)
	testutil.Equals(t, got, tag.Tags(nil))
	got, err = repo.List(ctx, other)
	testutil.Ok(t, err)
	testutil.Equals(t, got, tag.Tags{{Key: "a", Value: "other"}})
}

func TestRepositoryChangesRollBackWithOwningTransaction(t *testing.T) {
	t.Parallel()

	repo, s, ctx := newRepository(t)
	target := entity.NewOrderID().EntityUID()
	tx, err := s.Begin(ctx, true)
	testutil.Ok(t, err)
	txCtx := testContext{Context: ctx, tx: tx}

	changed, err := repo.Upsert(txCtx, target, tag.Tag{Key: "region", Value: "west"})
	testutil.Ok(t, err)
	testutil.IsTrue(t, changed)
	inside, err := repo.List(txCtx, target)
	testutil.Ok(t, err)
	testutil.Equals(t, inside, tag.Tags{{Key: "region", Value: "west"}})
	testutil.Ok(t, s.Rollback(tx))

	after, err := repo.List(ctx, target)
	testutil.Ok(t, err)
	testutil.Equals(t, after, tag.Tags(nil))
}

func TestRepositoryValidatesTargetsAndTags(t *testing.T) {
	t.Parallel()

	repo, s, ctx := newRepository(t)
	validDrink := entity.NewDrinkID().EntityUID()
	tests := []struct {
		name   string
		target cedar.EntityUID
		value  tag.Tag
	}{
		{name: "zero target", value: tag.Tag{Key: "valid"}},
		{name: "unsupported type", target: entity.NewAuditEntryID().EntityUID(), value: tag.Tag{Key: "valid"}},
		{name: "wrong id prefix", target: cedar.NewEntityUID(entity.TypeDrink, entity.NewMenuID().EntityUID().ID), value: tag.Tag{Key: "valid"}},
		{name: "empty key", target: validDrink, value: tag.Tag{}},
		{name: "untrimmed value", target: validDrink, value: tag.Tag{Key: "valid", Value: " west "}},
	}
	for _, tt := range tests { //nolint:paralleltest // Subtests share one repository and store.
		t.Run(tt.name, func(t *testing.T) {
			var gotErr error
			err := s.Write(ctx, func(tx *bstore.Tx) error {
				_, gotErr = repo.Upsert(testContext{Context: ctx, tx: tx}, tt.target, tt.value)
				return nil
			})
			testutil.Ok(t, err)
			testutil.ErrorIsInvalid(t, gotErr)
		})
	}

	err := s.Write(ctx, func(tx *bstore.Tx) error {
		_, err := repo.Remove(testContext{Context: ctx, tx: tx}, validDrink, " invalid ")
		return err
	})
	testutil.ErrorIsInvalid(t, err)
}

func newRepository(t *testing.T) (*tagging.Repository, *store.Store, testContext) {
	t.Helper()
	ctx := testContext{Context: context.Background()}
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "tagging.db"))
	testutil.Ok(t, err)
	t.Cleanup(func() { testutil.Ok(t, s.Close()) })
	tagging.RegisterSchema(ctx, s)
	return tagging.NewRepository(s), s, ctx
}

func upsert(t *testing.T, s *store.Store, ctx testContext, repo *tagging.Repository, target cedar.EntityUID, value tag.Tag) bool {
	t.Helper()
	changed := false
	err := s.Write(ctx, func(tx *bstore.Tx) error {
		var err error
		changed, err = repo.Upsert(testContext{Context: ctx, tx: tx}, target, value)
		return err
	})
	testutil.Ok(t, err)
	return changed
}

func remove(t *testing.T, s *store.Store, ctx testContext, repo *tagging.Repository, target cedar.EntityUID, key string) bool {
	t.Helper()
	changed := false
	err := s.Write(ctx, func(tx *bstore.Tx) error {
		var err error
		changed, err = repo.Remove(testContext{Context: ctx, tx: tx}, target, key)
		return err
	})
	testutil.Ok(t, err)
	return changed
}

func deleteTarget(t *testing.T, s *store.Store, ctx testContext, repo *tagging.Repository, target cedar.EntityUID) int {
	t.Helper()
	deleted := 0
	err := s.Write(ctx, func(tx *bstore.Tx) error {
		var err error
		deleted, err = repo.DeleteTarget(testContext{Context: ctx, tx: tx}, target)
		return err
	})
	testutil.Ok(t, err)
	return deleted
}
