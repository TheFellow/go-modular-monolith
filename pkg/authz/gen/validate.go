package main

import (
	"fmt"
	"os"
	"path/filepath"

	cedar "github.com/cedar-policy/cedar-go"
	cedarast "github.com/cedar-policy/cedar-go/x/exp/ast"
	"github.com/cedar-policy/cedar-go/x/exp/schema"
	"github.com/cedar-policy/cedar-go/x/exp/schema/resolved"
	"github.com/cedar-policy/cedar-go/x/exp/schema/validate"
)

func validateBasePolicies(repoRoot string) {
	directory := filepath.Join(repoRoot, "pkg", "authz")
	path := filepath.Join(directory, schemaFile)
	src, err := os.ReadFile(path)
	must(err)

	var s schema.Schema
	s.SetFilename(path)
	must(s.UnmarshalCedar(src))
	resolvedSchema, err := s.Resolve()
	must(err)
	must(validatePolicies(resolvedSchema, filepath.Join(directory, "base.cedar")))
}

func validatePolicies(s *resolved.Schema, path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	policies, err := cedar.NewPolicySetFromBytes(path, src)
	if err != nil {
		return err
	}
	validator := validate.New(s)
	for id, policy := range policies.All() {
		if err := validator.Policy(string(id), (*cedarast.Policy)(policy.AST())); err != nil {
			return fmt.Errorf("validate %s policy %s: %w", path, id, err)
		}
	}
	return nil
}
