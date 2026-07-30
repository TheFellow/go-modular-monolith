//nolint:paralleltest // tests mutate temporary module layouts and process working state.
package architecture_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/TheFellow/arch-lint/pkg/config"
	"github.com/TheFellow/arch-lint/pkg/linter"
)

func TestArchitectureRulesRejectForbiddenImports(t *testing.T) {
	repositoryRoot := repositoryRoot(t)
	fixtureRoot := t.TempDir()

	writeFixture(t, fixtureRoot, "go.mod", "module github.com/TheFellow/go-modular-monolith\n\ngo 1.26.5\n")
	configData, err := os.ReadFile(filepath.Join(repositoryRoot, ".arch-lint.yaml"))
	if err != nil {
		t.Fatalf("read repository architecture config: %v", err)
	}
	writeFixture(t, fixtureRoot, ".arch-lint.yaml", string(configData))

	// Import targets only need to exist: the fixture deliberately keeps them empty
	// so this test exercises architectural imports rather than application behavior.
	for _, target := range []string{
		"app",
		"main",
		"app/domains/drinks",
		"app/domains/drinks/internal/storage",
		"app/domains/drinks/internal/commands",
		"app/domains/drinks/internal/commands/nested",
		"app/domains/drinks/surfaces/cli",
		"app/domains/drinks/surfaces/gui",
		"app/domains/drinks/surfaces/tui",
		"app/domains/drinks/surfaces/web",
		"app/domains/ingredients/surfaces/gui",
		"app/domains/ingredients/internal/storage",
	} {
		writeFixture(t, fixtureRoot, filepath.Join(target, "target.go"), "package target\n")
	}

	writeImporter(t, fixtureRoot, "pkg/fyne/invalid", "app", "main")
	writeImporter(t, fixtureRoot, "pkg/tui/invalid", "app", "main")
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/gui/invalid",
		"app/domains/drinks/internal/storage",
		"app/domains/drinks/surfaces/cli",
		"app/domains/drinks/surfaces/tui",
		"main",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/gui/crossdomain",
		"app/domains/ingredients/surfaces/gui",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/gui/valid",
		"app/domains/drinks",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/tui/invalid-gui",
		"app/domains/drinks/surfaces/gui",
		"app/domains/ingredients/surfaces/gui",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/tui/valid",
		"app/domains/drinks",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/cli/invalid-gui",
		"app/domains/drinks/surfaces/gui",
		"app/domains/ingredients/surfaces/gui",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/cli/valid",
		"app/domains/drinks",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/web/invalid-gui",
		"app/domains/drinks/surfaces/gui",
		"app/domains/drinks/internal/storage",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/web/valid",
		"app/domains/drinks/surfaces/web",
	)
	writeImporterPackage(t, fixtureRoot, "app/domains/drinks", "target",
		"app/domains/drinks/internal/storage",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/internal/workflow/valid-internal",
		"app/domains/drinks/internal/storage",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/queries/valid-internal",
		"app/domains/drinks/internal/storage",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/handlers/valid-internal",
		"app/domains/drinks/internal/storage",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/service/invalid-internal",
		"app/domains/drinks/internal/storage",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/service/invalid-cross-domain-internal",
		"app/domains/ingredients/internal/storage",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/handlers/invalid",
		"app/domains/drinks",
		"app/domains/drinks/internal/commands",
		"app/domains/drinks/internal/commands/nested",
	)
	for _, layer := range []string{"authz", "events", "models"} {
		writeImporter(t, fixtureRoot, "app/domains/drinks/"+layer+"/invalid-internal",
			"app/domains/drinks/internal/storage",
		)
	}

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(fixtureRoot); err != nil {
		t.Fatalf("enter fixture module: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(workingDirectory); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	cfg, err := config.Load(".arch-lint.yaml")
	if err != nil {
		t.Fatalf("load architecture config: %v", err)
	}
	violations, err := linter.Run(cfg)
	if err != nil {
		t.Fatalf("run arch-lint against fixture module: %v", err)
	}

	want := []string{
		`arch-lint: [domain-internals-have-explicit-consumers] package "app/domains/drinks/authz/invalid-internal" imports "app/domains/drinks/internal/storage"`,
		`arch-lint: [domain-internals-have-explicit-consumers] package "app/domains/drinks/events/invalid-internal" imports "app/domains/drinks/internal/storage"`,
		`arch-lint: [domain-internals-have-explicit-consumers] package "app/domains/drinks/models/invalid-internal" imports "app/domains/drinks/internal/storage"`,
		`arch-lint: [domain-internals-have-explicit-consumers] package "app/domains/drinks/service/invalid-cross-domain-internal" imports "app/domains/ingredients/internal/storage"`,
		`arch-lint: [domain-internals-have-explicit-consumers] package "app/domains/drinks/service/invalid-internal" imports "app/domains/drinks/internal/storage"`,
		`arch-lint: [surfaces-are-bespoke] package "app/domains/drinks/surfaces/cli/invalid-gui" imports "app/domains/drinks/surfaces/gui"`,
		`arch-lint: [surfaces-are-bespoke] package "app/domains/drinks/surfaces/cli/invalid-gui" imports "app/domains/ingredients/surfaces/gui"`,
		`arch-lint: [surfaces-are-bespoke] package "app/domains/drinks/surfaces/gui/crossdomain" imports "app/domains/ingredients/surfaces/gui"`,
		`arch-lint: [handlers-no-commands] package "app/domains/drinks/handlers/invalid" imports "app/domains/drinks/internal/commands"`,
		`arch-lint: [handlers-no-commands] package "app/domains/drinks/handlers/invalid" imports "app/domains/drinks/internal/commands/nested"`,
		`arch-lint: [handlers-no-modules] package "app/domains/drinks/handlers/invalid" imports "app/domains/drinks"`,
		`arch-lint: [domain-internals-have-explicit-consumers] package "app/domains/drinks/surfaces/gui/invalid" imports "app/domains/drinks/internal/storage"`,
		`arch-lint: [surfaces-are-bespoke] package "app/domains/drinks/surfaces/gui/invalid" imports "app/domains/drinks/surfaces/cli"`,
		`arch-lint: [surfaces-are-bespoke] package "app/domains/drinks/surfaces/gui/invalid" imports "app/domains/drinks/surfaces/tui"`,
		`arch-lint: [gui-surfaces-no-composition-or-tui] package "app/domains/drinks/surfaces/gui/invalid" imports "main"`,
		`arch-lint: [presentation-toolkits-no-application] package "pkg/fyne/invalid" imports "app"`,
		`arch-lint: [presentation-toolkits-no-application] package "pkg/fyne/invalid" imports "main"`,
		`arch-lint: [presentation-toolkits-no-application] package "pkg/tui/invalid" imports "app"`,
		`arch-lint: [presentation-toolkits-no-application] package "pkg/tui/invalid" imports "main"`,
		`arch-lint: [surfaces-are-bespoke] package "app/domains/drinks/surfaces/tui/invalid-gui" imports "app/domains/drinks/surfaces/gui"`,
		`arch-lint: [surfaces-are-bespoke] package "app/domains/drinks/surfaces/tui/invalid-gui" imports "app/domains/ingredients/surfaces/gui"`,
		`arch-lint: [surfaces-are-bespoke] package "app/domains/drinks/surfaces/web/invalid-gui" imports "app/domains/drinks/surfaces/gui"`,
		`arch-lint: [domain-internals-have-explicit-consumers] package "app/domains/drinks/surfaces/web/invalid-gui" imports "app/domains/drinks/internal/storage"`,
	}
	got := make([]string, 0, len(violations))
	for _, violation := range violations {
		got = append(got, violation.String())
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("unexpected architecture violations:\n got: %q\nwant: %q", got, want)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate architecture test source")
	}
	return filepath.Dir(filepath.Dir(filename))
}

func writeImporter(t *testing.T, root, directory string, imports ...string) {
	t.Helper()
	writeImporterPackage(t, root, directory, "fixture", imports...)
}

func writeImporterPackage(t *testing.T, root, directory, packageName string, imports ...string) {
	t.Helper()
	var source strings.Builder
	fmt.Fprintf(&source, "package %s\n\nimport (\n", packageName)
	for _, importPath := range imports {
		fmt.Fprintf(&source, "\t_ %q\n", "github.com/TheFellow/go-modular-monolith/"+importPath)
	}
	source.WriteString(")\n")
	writeFixture(t, root, filepath.Join(directory, "fixture.go"), source.String())
}

func writeFixture(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
}
