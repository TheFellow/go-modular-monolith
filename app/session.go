package app

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

// Session binds an application to one authenticated context for a persistent
// client such as the TUI.
type Session struct {
	*App
	ctx context.Context
}

func NewSession(ctx context.Context, application *App) *Session {
	return &Session{App: application, ctx: ctx}
}

func (s *Session) Context() *middleware.Context {
	return middleware.NewContext(s.ctx)
}
