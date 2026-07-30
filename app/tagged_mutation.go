package app

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

// TaggableEntity is the result contract for an application mutation whose
// target supports tags. Domain models satisfy this contract without depending
// on a presentation surface.
type TaggableEntity interface {
	EntityUID() cedar.EntityUID
	SetTags(tag.Tags)
}

// RunTaggedMutation executes a domain mutation and an optional complete tag-set
// replacement in one transaction. A nil desired set preserves existing tags;
// a non-nil empty set clears them. Validation happens before the transaction so
// invalid tags cannot execute the domain mutation.
//
// This application-level composition is shared by presentation surfaces while
// their parsing, form state, feedback, and interaction remain framework-native.
func RunTaggedMutation[T TaggableEntity](
	application *App,
	ctx *middleware.Context,
	desired *tag.Tags,
	mutate func(*middleware.Context) (T, error),
) (T, error) {
	var zero T
	if desired == nil {
		return mutate(ctx)
	}
	if err := desired.Validate(); err != nil {
		return zero, err
	}

	var result T
	compose := func(txCtx *middleware.Context) error {
		var err error
		result, err = mutate(txCtx)
		if err != nil {
			return err
		}
		replaced, err := application.Tags.Replace(txCtx, result.EntityUID(), *desired)
		if err != nil {
			return err
		}
		result.SetTags(replaced.Tags)
		return nil
	}

	// Participate in a caller-owned transaction when this composition is one
	// step of a larger application workflow. The caller retains commit and
	// rollback ownership, matching the domain command pipeline's convention.
	if tx, ok := ctx.Transaction(); ok && tx != nil {
		if err := compose(ctx); err != nil {
			return zero, err
		}
		return result, nil
	}

	err := application.Store.Write(ctx, func(tx *bstore.Tx) error {
		return compose(ctx.WithTransaction(tx))
	})
	if err != nil {
		return zero, err
	}
	return result, nil
}
