package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func requireMiddlewareContext(ctx context.Context) (*middleware.Context, error) {
	mctx, ok := ctx.(*middleware.Context)
	if !ok {
		return nil, fmt.Errorf("expected middleware context")
	}
	return mctx, nil
}

func writeJSON(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = w.Write(b)
	return err
}
