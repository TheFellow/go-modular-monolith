package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/urfave/cli/v3"
)

var (
	JSONFlag     cli.Flag = &cli.BoolFlag{Name: "json", Usage: "Output JSON"}
	TemplateFlag cli.Flag = &cli.BoolFlag{Name: "template", Usage: "Print JSON template and exit"}
	StdinFlag    cli.Flag = &cli.BoolFlag{Name: "stdin", Usage: "Read JSON from stdin"}
	FileFlag     cli.Flag = &cli.StringFlag{Name: "file", Usage: "Read JSON from file"}
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
