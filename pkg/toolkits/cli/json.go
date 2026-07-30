package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	urfave "github.com/urfave/cli/v3"
)

var (
	JSONFlag     urfave.Flag = &urfave.BoolFlag{Name: "json", Usage: "Output JSON"}
	TemplateFlag urfave.Flag = &urfave.BoolFlag{Name: "template", Usage: "Print JSON template and exit"}
	StdinFlag    urfave.Flag = &urfave.BoolFlag{Name: "stdin", Usage: "Read JSON from stdin"}
	FileFlag     urfave.Flag = &urfave.StringFlag{Name: "file", Usage: "Read JSON from file"}
)

func WriteJSON(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = w.Write(b)
	return err
}

func ReadJSONInput[T any](cmd *urfave.Command) (T, error) {
	var zero T

	fromStdin := cmd.Bool("stdin")
	fromFile := strings.TrimSpace(cmd.String("file"))
	if fromStdin && fromFile != "" {
		return zero, fmt.Errorf("set only one of --stdin or --file")
	}
	if !fromStdin && fromFile == "" {
		return zero, fmt.Errorf("missing input: set --stdin or --file (or use --template)")
	}

	var r io.Reader
	if fromStdin {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return zero, err
		}
		if len(bytes.TrimSpace(b)) == 0 {
			return zero, fmt.Errorf("stdin is empty")
		}
		r = bytes.NewReader(b)
	} else {
		f, err := os.Open(fromFile)
		if err != nil {
			return zero, err
		}
		defer func() { _ = f.Close() }()
		r = f
	}

	var result T
	if err := json.NewDecoder(r).Decode(&result); err != nil {
		return zero, err
	}
	return result, nil
}
