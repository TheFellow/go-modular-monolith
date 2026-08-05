package actions_test

import (
	"context"
	stderrors "errors"
	"testing"

	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func allow(context.Context) error { return nil }

func deny(context.Context) error { return apperrors.Permissionf("not permitted") }

func TestEvaluatePermissionInheritanceAndOverrides(t *testing.T) {
	t.Parallel()

	states, err := actions.Evaluate(context.Background(), actions.Group{
		Permission: actions.Require(deny),
		Controls: []actions.Control{
			{ID: "edit"},
			{ID: "publish", Permission: actions.Require(allow)},
			{ID: "help", Permission: actions.Public()},
		},
		Groups: []actions.Group{{
			Permission: actions.Require(allow),
			Controls:   []actions.Control{{ID: "archive"}},
		}},
	})
	testutil.Ok(t, err)
	testutil.Equals(t, states, []actions.State{
		{ID: "edit", Visible: false, Enabled: false},
		{ID: "publish", Visible: true, Enabled: true},
		{ID: "help", Visible: true, Enabled: true},
		{ID: "archive", Visible: true, Enabled: true},
	})
}

func TestEvaluateAuthorizedUnavailableControlIsVisibleAndDisabled(t *testing.T) {
	t.Parallel()

	secondCalled := false
	states, err := actions.Evaluate(context.Background(), actions.Group{
		Permission: actions.Require(allow),
		Controls: []actions.Control{{
			ID: "publish",
			Conditions: []actions.Condition{
				func(context.Context) (bool, string, error) { return false, "save changes before publishing", nil },
				func(context.Context) (bool, string, error) {
					secondCalled = true
					return true, "", nil
				},
			},
		}},
	})
	testutil.Ok(t, err)
	testutil.Equals(t, states, []actions.State{{
		ID:             "publish",
		Visible:        true,
		Enabled:        false,
		DisabledReason: "save changes before publishing",
	}})
	testutil.IsFalse(t, secondCalled)
}

func TestEvaluateDeniedControlDoesNotEvaluateConditions(t *testing.T) {
	t.Parallel()

	conditionCalled := false
	states, err := actions.Evaluate(context.Background(), actions.Group{
		Controls: []actions.Control{{
			ID:         "delete",
			Permission: actions.Require(deny),
			Conditions: []actions.Condition{func(context.Context) (bool, string, error) {
				conditionCalled = true
				return true, "", nil
			}},
		}},
	})
	testutil.Ok(t, err)
	testutil.Equals(t, states, []actions.State{{ID: "delete", Visible: false, Enabled: false}})
	testutil.IsFalse(t, conditionCalled)
}

func TestEvaluateReturnsAuthorizationEvaluationError(t *testing.T) {
	t.Parallel()

	want := stderrors.New("policy service unavailable")
	_, err := actions.Evaluate(context.Background(), actions.Group{
		Controls: []actions.Control{{
			ID: "publish",
			Permission: actions.Require(func(context.Context) error {
				return want
			}),
		}},
	})
	testutil.ErrorIf(t, err == nil, "expected an error")
	testutil.ErrorIf(t, !stderrors.Is(err, want), "error = %v, want wrapping %v", err, want)
}

func TestEvaluateReturnsConditionError(t *testing.T) {
	t.Parallel()

	want := stderrors.New("entity failed to load")
	_, err := actions.Evaluate(context.Background(), actions.Group{
		Controls: []actions.Control{{
			ID: "publish",
			Conditions: []actions.Condition{func(context.Context) (bool, string, error) {
				return false, "", want
			}},
		}},
	})
	testutil.ErrorIf(t, err == nil, "expected an error")
	testutil.ErrorIf(t, !stderrors.Is(err, want), "error = %v, want wrapping %v", err, want)
}

func TestEvaluateRejectsInvalidDeclarations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		group actions.Group
	}{
		{name: "empty ID", group: actions.Group{Controls: []actions.Control{{}}}},
		{name: "duplicate nested ID", group: actions.Group{
			Controls: []actions.Control{{ID: "save"}},
			Groups:   []actions.Group{{Controls: []actions.Control{{ID: "save"}}}},
		}},
		{name: "duplicate sibling ID", group: actions.Group{Groups: []actions.Group{
			{Controls: []actions.Control{{ID: "save"}}},
			{Controls: []actions.Control{{ID: "save"}}},
		}}},
		{name: "nil required permission", group: actions.Group{
			Permission: actions.Require(nil),
			Controls:   []actions.Control{{ID: "save"}},
		}},
		{name: "nil condition", group: actions.Group{
			Controls: []actions.Control{{ID: "save", Conditions: []actions.Condition{nil}}},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := actions.Evaluate(context.Background(), tt.group)
			testutil.ErrorIf(t, err == nil, "expected an error")
		})
	}
}

func TestEvaluatePassesContextToChecks(t *testing.T) {
	t.Parallel()

	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "actor")
	var authorizationValue, conditionValue any
	states, err := actions.Evaluate(ctx, actions.Group{
		Permission: actions.Require(func(ctx context.Context) error {
			authorizationValue = ctx.Value(contextKey{})
			return nil
		}),
		Controls: []actions.Control{{
			ID: "save",
			Conditions: []actions.Condition{func(ctx context.Context) (bool, string, error) {
				conditionValue = ctx.Value(contextKey{})
				return true, "", nil
			}},
		}},
	})
	testutil.Ok(t, err)
	testutil.Equals(t, states, []actions.State{{ID: "save", Visible: true, Enabled: true}})
	testutil.Equals(t, authorizationValue, "actor")
	testutil.Equals(t, conditionValue, "actor")
}
