package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type taggedResult interface {
	middleware.CedarEntity
	SetTags(tag.Tags)
}

func withTags[In middleware.CedarEntity, Out taggedResult](edits []tag.Edit, handle middleware.CommandHandler[In, Out]) middleware.CommandHandler[In, Out] {
	return func(ctx *middleware.Context, in In) (Out, error) {
		var zero Out
		if len(edits) > 1 {
			return zero, errors.Invalidf("only one tag replacement is allowed")
		}
		if len(edits) == 0 || edits[0].Desired == nil {
			return handle(ctx, in)
		}
		edit := edits[0]
		if err := edit.Desired.Validate(); err != nil {
			return zero, err
		}
		result, err := handle(ctx, in)
		if err != nil {
			return zero, err
		}
		ctx.AddEvent(events.TagsReplaced{Replacement: tag.Replacement{Edit: edit, Entity: result.CedarEntity(), TagAction: authz.ActionTag, UntagAction: authz.ActionUntag}})
		result.SetTags(edit.Desired.Sorted())
		return result, nil
	}
}
