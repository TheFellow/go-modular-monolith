package main

import (
	"context"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	pkglog "github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
)

//nolint:paralleltest // exercises independent process lifecycles against one database file.
func TestCLIAndComposedTUIShareTagContracts(t *testing.T) {
	repository, err := filepath.Abs("../..")
	testutil.Ok(t, err)
	workingDirectory := t.TempDir()
	binary := testutil.ExecutablePath(workingDirectory, "mixology")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "./main/cli")
	build.Dir = repository
	if output, buildErr := build.CombinedOutput(); buildErr != nil {
		t.Fatalf("build CLI: %v\n%s", buildErr, output)
	}
	runCLI := func(args ...string) string {
		t.Helper()
		command := exec.CommandContext(t.Context(), binary, append([]string{"--log-level", "error"}, args...)...)
		command.Dir = workingDirectory
		output, runErr := command.CombinedOutput()
		if runErr != nil {
			t.Fatalf("CLI %v: %v\n%s", args, runErr, output)
		}
		return string(output)
	}

	ingredientID := strings.TrimSpace(runCLI("ingredients", "create", "--category", "other", "--unit", "oz", "Cross-surface ingredient"))
	runCLI("tags", "add", ingredientID, "surface=cli")

	ctx := authn.ToContext(context.Background(), authn.Owner())
	ctx = pkglog.ToContext(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)))
	database, err := store.Open(ctx, filepath.Join(workingDirectory, defaultDatabasePath))
	testutil.Ok(t, err)
	application := app.New(ctx, app.Config{Store: database})
	driver := tuitest.NewDriver(t, NewApp(app.NewSession(ctx, application)))
	driver.Resize(100, 40)
	driver.Press("7")
	for range 3 {
		driver.Press("down")
	}
	driver.Press("enter")
	driver.Press("surface=cli")
	driver.Press("ctrl+s")
	driver.RequireText("Show exact tag", ingredientID, "surface=cli")
	driver.Press("esc")
	driver.Press("esc")
	driver.Press("2")
	driver.RequireText("Mixology > Ingredients")
	driver.Press("t")
	driver.Press("ctrl+u")
	for _, value := range "surface=tui" {
		driver.Press(string(value))
	}
	driver.Press("ctrl+s")
	driver.RequireText("surface=tui")
	testutil.Ok(t, application.Close())

	output := runCLI("tags", "list", ingredientID)
	testutil.StringContains(t, output, ingredientID+": surface=tui")
}
