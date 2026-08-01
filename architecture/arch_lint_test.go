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
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestArchitectureRulesRejectForbiddenImports(t *testing.T) {
	repositoryRoot := repositoryRoot(t)
	fixtureRoot := t.TempDir()

	writeFixture(t, fixtureRoot, "go.mod", "module github.com/TheFellow/go-modular-monolith\n\ngo 1.26.5\n")
	configData, err := os.ReadFile(filepath.Join(repositoryRoot, ".arch-lint.yaml"))
	if err != nil {
		testutil.ErrorIf(t, true, "read repository architecture config: %v", err)
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
		"app/domains/drinks/authz",
		"app/domains/drinks/events",
		"app/domains/drinks/events/nested",
		"app/domains/drinks/models",
		"app/domains/drinks/queries",
		"app/domains/drinks/surfaces/cli",
		"app/domains/drinks/surfaces/gui",
		"app/domains/drinks/surfaces/tui",
		"app/domains/drinks/surfaces/web",
		"app/domains/ingredients/surfaces/gui",
		"app/domains/ingredients/authz",
		"app/domains/ingredients/authz/roles",
		"app/domains/ingredients/events",
		"app/domains/ingredients/events/nested",
		"app/domains/ingredients/models",
		"app/domains/ingredients/queries",
		"app/domains/ingredients/internal/storage",
		"pkg/toolkits/cli",
		"pkg/toolkits/gui",
		"pkg/toolkits/tui",
	} {
		writeFixture(t, fixtureRoot, filepath.Join(target, "target.go"), "package target\n")
	}

	writeImporter(t, fixtureRoot, "pkg/toolkits/gui/invalid", "app", "main")
	writeImporter(t, fixtureRoot, "pkg/toolkits/tui/invalid", "app", "main")
	writeImporter(t, fixtureRoot, "pkg/toolkits/cli/invalid", "app", "main")
	writeImporter(t, fixtureRoot, "pkg/toolkits/gui/cross-toolkit", "pkg/toolkits/tui")
	writeImporter(t, fixtureRoot, "pkg/toolkits/tui/cross-toolkit", "pkg/toolkits/cli")
	writeImporter(t, fixtureRoot, "pkg/toolkits/cli/cross-toolkit", "pkg/toolkits/gui")
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
		"app/domains/drinks", "pkg/toolkits/gui",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/gui/wrong-toolkit", "pkg/toolkits/tui")
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/tui/invalid-gui",
		"app/domains/drinks/surfaces/gui",
		"app/domains/ingredients/surfaces/gui",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/tui/valid",
		"app/domains/drinks", "pkg/toolkits/tui",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/tui/wrong-toolkit", "pkg/toolkits/gui")
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/cli/invalid-gui",
		"app/domains/drinks/surfaces/gui",
		"app/domains/ingredients/surfaces/gui",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/cli/valid",
		"app/domains/drinks", "pkg/toolkits/cli",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/surfaces/cli/wrong-toolkit", "pkg/toolkits/gui")
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
	writeImporter(t, fixtureRoot, "app/domains/drinks/service/valid-public-contracts",
		"app/domains/ingredients/events",
		"app/domains/ingredients/events/nested",
		"app/domains/ingredients/models",
		"app/domains/ingredients/queries",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/service/invalid-cross-domain-authz",
		"app/domains/ingredients/authz",
		"app/domains/ingredients/authz/roles",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/service/valid-own-authz",
		"app/domains/drinks/authz",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/internal/commands/valid-own-events",
		"app/domains/drinks/events",
		"app/domains/drinks/events/nested",
	)
	writeImporter(t, fixtureRoot, "app/domains/drinks/internal/commands/invalid-cross-domain-events",
		"app/domains/ingredients/events",
		"app/domains/ingredients/events/nested",
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
		testutil.ErrorIf(t, true, "get working directory: %v", err)
	}
	if err := os.Chdir(fixtureRoot); err != nil {
		testutil.ErrorIf(t, true, "enter fixture module: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(workingDirectory); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	cfg, err := config.Load(".arch-lint.yaml")
	if err != nil {
		testutil.ErrorIf(t, true, "load architecture config: %v", err)
	}
	violations, err := linter.Run(cfg)
	if err != nil {
		testutil.ErrorIf(t, true, "run arch-lint against fixture module: %v", err)
	}

	want := []string{
		`arch-lint: [commands-emit-own-domain-events] package "app/domains/drinks/internal/commands/invalid-cross-domain-events" imports "app/domains/ingredients/events"`,
		`arch-lint: [commands-emit-own-domain-events] package "app/domains/drinks/internal/commands/invalid-cross-domain-events" imports "app/domains/ingredients/events/nested"`,
		`arch-lint: [domain-authz-is-private] package "app/domains/drinks/service/invalid-cross-domain-authz" imports "app/domains/ingredients/authz"`,
		`arch-lint: [domain-authz-is-private] package "app/domains/drinks/service/invalid-cross-domain-authz" imports "app/domains/ingredients/authz/roles"`,
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
		`arch-lint: [surfaces-no-composition] package "app/domains/drinks/surfaces/gui/invalid" imports "main"`,
		`arch-lint: [surfaces-use-matching-toolkit] package "app/domains/drinks/surfaces/gui/wrong-toolkit" imports "pkg/toolkits/tui"`,
		`arch-lint: [surfaces-use-matching-toolkit] package "app/domains/drinks/surfaces/tui/wrong-toolkit" imports "pkg/toolkits/gui"`,
		`arch-lint: [surfaces-use-matching-toolkit] package "app/domains/drinks/surfaces/cli/wrong-toolkit" imports "pkg/toolkits/gui"`,
		`arch-lint: [presentation-toolkits-are-independent] package "pkg/toolkits/gui/cross-toolkit" imports "pkg/toolkits/tui"`,
		`arch-lint: [presentation-toolkits-are-independent] package "pkg/toolkits/tui/cross-toolkit" imports "pkg/toolkits/cli"`,
		`arch-lint: [presentation-toolkits-are-independent] package "pkg/toolkits/cli/cross-toolkit" imports "pkg/toolkits/gui"`,
		`arch-lint: [presentation-toolkits-no-application] package "pkg/toolkits/cli/invalid" imports "app"`,
		`arch-lint: [presentation-toolkits-no-application] package "pkg/toolkits/cli/invalid" imports "main"`,
		`arch-lint: [presentation-toolkits-no-application] package "pkg/toolkits/gui/invalid" imports "app"`,
		`arch-lint: [presentation-toolkits-no-application] package "pkg/toolkits/gui/invalid" imports "main"`,
		`arch-lint: [presentation-toolkits-no-application] package "pkg/toolkits/tui/invalid" imports "app"`,
		`arch-lint: [presentation-toolkits-no-application] package "pkg/toolkits/tui/invalid" imports "main"`,
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
		testutil.ErrorIf(t, true, "unexpected architecture violations:\n got: %q\nwant: %q", got, want)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		testutil.ErrorIf(t, true, "%v", "locate architecture test source")
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
		testutil.ErrorIf(t, true, "create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		testutil.ErrorIf(t, true, "write fixture %s: %v", name, err)
	}
}
