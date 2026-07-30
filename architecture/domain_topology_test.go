package architecture_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type domainProfile struct {
	rootPackages     []string
	internalPackages []string
	surfacePackages  []string
}

var (
	operationalDomain = domainProfile{
		rootPackages:     []string{"authz", "events", "handlers", "internal", "models", "queries", "surfaces"},
		internalPackages: []string{"availability", "commands", "dao"},
		surfacePackages:  []string{"cli", "gui", "tui"},
	}
	auditDomain = domainProfile{
		rootPackages:     []string{"authz", "internal", "models", "queries", "surfaces"},
		internalPackages: []string{"dao"},
		surfacePackages:  []string{"cli", "gui", "tui"},
	}
	taggingDomain = domainProfile{
		rootPackages:    []string{"authz", "surfaces"},
		surfacePackages: []string{"gui"},
	}
)

func TestDomainPackageTopology(t *testing.T) {
	domainsRoot := filepath.Join(repositoryRoot(t), "app", "domains")
	domains, err := os.ReadDir(domainsRoot)
	if err != nil {
		t.Fatalf("read domains directory: %v", err)
	}

	for _, domain := range domains {
		if !domain.IsDir() {
			continue
		}

		domain := domain
		t.Run(domain.Name(), func(t *testing.T) {
			domainRoot := filepath.Join(domainsRoot, domain.Name())
			if _, err := os.Stat(filepath.Join(domainRoot, "module.go")); err != nil {
				t.Errorf("domain root must define its composition boundary in module.go: %v", err)
			}

			for _, violation := range domainTopologyViolations(domainRoot, profileForDomain(domain.Name())) {
				t.Error(violation)
			}
		})
	}
}

func TestOperationalDomainProfileRejectsNovelPackageLayers(t *testing.T) {
	domainRoot := t.TempDir()
	for _, packagePath := range []string{
		"models",
		"models/generated_test.go",
		"internal/availability",
		"surfaces/gui",
		"services",
		"utils",
		"internal/repositories",
	} {
		if filepath.Ext(packagePath) == ".go" {
			writeFixture(t, domainRoot, packagePath, "package models_test\n")
			continue
		}
		writeFixture(t, domainRoot, filepath.Join(packagePath, "package.go"), "package fixture\n")
	}

	got := domainTopologyViolations(domainRoot, operationalDomain)
	want := []string{
		`package "internal/repositories" is outside the domain profile; allowed root packages are authz, events, handlers, internal, models, queries, surfaces`,
		`package "services" is outside the domain profile; allowed root packages are authz, events, handlers, internal, models, queries, surfaces`,
		`package "utils" is outside the domain profile; allowed root packages are authz, events, handlers, internal, models, queries, surfaces`,
	}
	if !slices.Equal(got, want) {
		t.Fatalf("unexpected topology violations:\n got: %q\nwant: %q", got, want)
	}
}

func profileForDomain(name string) domainProfile {
	switch name {
	case "audit":
		return auditDomain
	case "tagging":
		return taggingDomain
	default:
		return operationalDomain
	}
}

func domainTopologyViolations(domainRoot string, profile domainProfile) []string {
	var violations []string
	err := filepath.WalkDir(domainRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() || path == domainRoot {
			return nil
		}

		relative, err := filepath.Rel(domainRoot, path)
		if err != nil {
			return err
		}
		if !containsGoPackage(path) {
			return nil
		}

		parts := strings.Split(filepath.ToSlash(relative), "/")
		allowed := false
		switch len(parts) {
		case 1:
			allowed = slices.Contains(profile.rootPackages, parts[0])
		case 2:
			switch parts[0] {
			case "internal":
				allowed = slices.Contains(profile.internalPackages, parts[1])
			case "surfaces":
				allowed = slices.Contains(profile.surfacePackages, parts[1])
			}
		}
		if !allowed {
			violations = append(violations, fmt.Sprintf(
				"package %q is outside the domain profile; allowed root packages are %s",
				filepath.ToSlash(relative), strings.Join(profile.rootPackages, ", "),
			))
		}
		return nil
	})
	if err != nil {
		return append(violations, fmt.Sprintf("inspect domain packages: %v", err))
	}
	slices.Sort(violations)
	return violations
}

func containsGoPackage(directory string) bool {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			return true
		}
	}
	return false
}
