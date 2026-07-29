package architecture_test

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/TheFellow/arch-lint/pkg/config"
	"github.com/TheFellow/arch-lint/pkg/linter"
)

func TestFyneArchitectureRulesRejectForbiddenImports(t *testing.T) {
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
		"app/domains/drinks/surfaces/cli",
		"app/domains/drinks/surfaces/fyne",
		"app/domains/drinks/surfaces/tui",
		"app/domains/ingredients/surfaces/fyne",
	} {
		writeFixture(t, fixtureRoot, filepath.Join(target, "target.go"), "package target\n")
	}

	writeImporter(t, fixtureRoot, "pkg/fyne/invalid", "app", "main")
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/fyne/invalid",
		"app/domains/drinks/internal/storage",
		"app/domains/drinks/surfaces/cli",
		"app/domains/drinks/surfaces/tui",
		"main",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/fyne/crossdomain",
		"app/domains/ingredients/surfaces/fyne",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/fyne/valid",
		"app/domains/drinks",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/tui/invalid-fyne",
		"app/domains/drinks/surfaces/fyne",
		"app/domains/ingredients/surfaces/fyne",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/tui/valid",
		"app/domains/drinks",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/cli/invalid-fyne",
		"app/domains/drinks/surfaces/fyne",
		"app/domains/ingredients/surfaces/fyne",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/cli/valid",
		"app/domains/drinks",
	)

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
		`arch-lint: [cli-surfaces-are-bespoke] package "app/domains/drinks/surfaces/cli/invalid-fyne" imports "app/domains/drinks/surfaces/fyne"`,
		`arch-lint: [cli-surfaces-are-bespoke] package "app/domains/drinks/surfaces/cli/invalid-fyne" imports "app/domains/ingredients/surfaces/fyne"`,
		`arch-lint: [fyne-surfaces-no-cross-domain-surfaces] package "app/domains/drinks/surfaces/fyne/crossdomain" imports "app/domains/ingredients/surfaces/fyne"`,
		`arch-lint: [fyne-surfaces-use-public-domain-api] package "app/domains/drinks/surfaces/fyne/invalid" imports "app/domains/drinks/internal/storage"`,
		`arch-lint: [fyne-surfaces-use-public-domain-api] package "app/domains/drinks/surfaces/fyne/invalid" imports "app/domains/drinks/surfaces/cli"`,
		`arch-lint: [fyne-surfaces-use-public-domain-api] package "app/domains/drinks/surfaces/fyne/invalid" imports "app/domains/drinks/surfaces/tui"`,
		`arch-lint: [fyne-surfaces-use-public-domain-api] package "app/domains/drinks/surfaces/fyne/invalid" imports "main"`,
		`arch-lint: [fyne-toolkit-no-application] package "pkg/fyne/invalid" imports "app"`,
		`arch-lint: [fyne-toolkit-no-application] package "pkg/fyne/invalid" imports "main"`,
		`arch-lint: [tui-surfaces-are-bespoke] package "app/domains/drinks/surfaces/tui/invalid-fyne" imports "app/domains/drinks/surfaces/fyne"`,
		`arch-lint: [tui-surfaces-are-bespoke] package "app/domains/drinks/surfaces/tui/invalid-fyne" imports "app/domains/ingredients/surfaces/fyne"`,
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
	source := "package fixture\n\nimport (\n"
	for _, importPath := range imports {
		source += "\t_ \"github.com/TheFellow/go-modular-monolith/" + importPath + "\"\n"
	}
	source += ")\n"
	writeFixture(t, root, filepath.Join(directory, "fixture.go"), source)
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
