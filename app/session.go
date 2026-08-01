package app

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

// Session binds an application to one authenticated context for a persistent
// client such as the TUI.
type Session struct {
	*App
	ctx context.Context
}

// ReplaceTags replaces an entity's complete tag set in this session's context.
func (s *Session) ReplaceTags(target cedar.EntityUID, desired tag.Tags) (tag.Tags, error) {
	result, err := s.Tags.Replace(s.Context(), target, desired)
	return result.Tags, err
}

func NewSession(ctx context.Context, application *App) *Session {
	return &Session{App: application, ctx: ctx}
}

func (s *Session) Context() *middleware.Context {
	return middleware.NewContext(s.ctx)
}

// ContextFrom creates an operation context from a cancellable descendant of
// this session's authenticated context.
func (s *Session) ContextFrom(ctx context.Context) *middleware.Context {
	if ctx == nil {
		return middleware.NewContext(s.ctx)
	}
	derived, cancel := context.WithCancel(s.ctx)
	if deadline, ok := ctx.Deadline(); ok {
		derived, cancel = context.WithDeadline(s.ctx, deadline)
	}
	context.AfterFunc(ctx, cancel)
	return middleware.NewContext(derived)
}
