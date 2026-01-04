package config

import (
	"fmt"
	"os"
	"strings"

	_ "embed"
	"github.com/goccy/go-yaml"
	"github.com/xeipuuv/gojsonschema"
)

//go:embed schema.yml
var schemaData []byte

type Config struct {
	IncludeTests bool   `yaml:"include_tests"`
	Specs        []Spec `yaml:"specs"`
}

type Spec struct {
	Name     string   `yaml:"name"`
	Packages Packages `yaml:"packages"`
	Rules    Rules    `yaml:"rules"`
}

type Rules struct {
	Forbid []string `yaml:"forbid"`
	Except []string `yaml:"except"`
	Exempt []string `yaml:"exempt"`
}

type Packages struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	if err := validateSchema(data); err != nil {
		return nil, fmt.Errorf("config schema validation failed: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if len(cfg.Specs) == 0 {
		return nil, fmt.Errorf("config must contain at least one spec")
	}
	for _, r := range cfg.Specs {
		if len(r.Packages.Include) == 0 {
			return nil, fmt.Errorf("rule '%s' must specify 'packages'", r.Name)
		}
		if len(r.Rules.Forbid) == 0 {
			return nil, fmt.Errorf("rule '%s' must specify 'forbid' rules", r.Name)
		}
	}
	return &cfg, nil
}

func validateSchema(data []byte) error {
	schemaJSON, err := yaml.YAMLToJSON(schemaData)
	if err != nil {
		return err
	}
	docJSON, err := yaml.YAMLToJSON(data)
	if err != nil {
		return err
	}
	schemaLoader := gojsonschema.NewBytesLoader(schemaJSON)
	docLoader := gojsonschema.NewBytesLoader(docJSON)
	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return err
	}
	if !result.Valid() {
		var sb strings.Builder
		for i, e := range result.Errors() {
			if i > 0 {
				sb.WriteString("; ")
			}
			sb.WriteString(e.String())
		}
		return fmt.Errorf(sb.String())
	}
	return nil
}
