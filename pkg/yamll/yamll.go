package yamll

import (
	"fmt"
	"log/slog"

	"github.com/nikhilsbhat/yamll/pkg/errors"
)

// YamlData holds information of yaml file and its dependency tree.
type YamlData struct {
	Root       bool          `json:"root,omitempty" yaml:"root,omitempty"`
	Merged     bool          `json:"merged,omitempty" yaml:"merged,omitempty"`
	File       string        `json:"file,omitempty" yaml:"file,omitempty"`
	DataRaw    string        `json:"data_raw,omitempty" yaml:"data_raw,omitempty"`
	Dependency []*Dependency `json:"dependency,omitempty" yaml:"dependency,omitempty"`
}

// Config holds the information of yaml files to be parsed.
type Config struct {
	Root      bool          `json:"root,omitempty" yaml:"root,omitempty"`
	Effective bool          `json:"effective,omitempty" yaml:"effective,omitempty"`
	Limiter   string        `json:"limiter,omitempty" yaml:"limiter,omitempty"`
	LogLevel  string        `json:"log_level,omitempty" yaml:"log_level,omitempty"`
	Files     []*Dependency `json:"files,omitempty" yaml:"files,omitempty"`
	log       *slog.Logger
}

// YamlRoutes holds a map of YamlData, representing a dependency tree.
type YamlRoutes map[string]*YamlData

// Yaml is a string representation of YAML content.
type Yaml string

// Yaml identifies the YAML imports and merges them to create a single comprehensive YAML file.
// These imports work in a manner similar to importing libraries in a programming language.
// It searches for the imports defined in any of the following (comments that start with ##++ in your YAML definition).
// Supports importing from various sources including local files, URLs, and Git.
// Sample imports look like:
//
//	##++internal/fixtures/base2.yaml
//	##++https://run.mocky.io/v3/92e08b25-dd1f-4dd0-bc55-9649b5b896c9
//	##++git+https://github.com/nikhilsbhat/yamll@main?path=internal/fixtures/base.yaml
//	##++git+ssh://git@github.com:nikhilsbhat/yamll@main?path=internal/fixtures/base.yaml
//	##++git+ssh://git@github.com:nikhilsbhat/yamll@v0.2.5?path=internal/fixtures/base.yaml
//	##++https://test.com/test.yaml;{"user_name":"${username}","password":"${pass}","ca_content":"${ca_content}"}
//
// The parameters necessary for authenticating the remote server in URL/GIT based imports should be defined as shown in the example above.
// All supported parameters can found under Auth.
//
// Authentication parameters, which cannot be directly specified in imports for security reasons, can be replaced with environment variables.
// To use this feature, define the parameter exposed as an environment variable as $VARIABLE_NAME, as shown in the last example.
//
// Breakdown of git repo based import:
// http based url: ##++git+https://github.com/<org_name>/<repo_name>@<branch/tag>?path=<path/to/file.yaml> ex: ##++git+https://github.com/nikhilsbhat/yamll@main?path=internal/fixtures/base.yaml.
// ssh based url ##++git+ssh://git@github.com:<org_name>/<repo_name>@<branch/tag>?path=<path/to/file.yaml> ex: ##++git+ssh://git@github.com:nikhilsbhat/yamll@main?path=internal/fixtures/base.yaml.
//
//nolint:lll
func (cfg *Config) Yaml() (Yaml, error) {
	dependencyRoutes, err := cfg.ResolveDependencies(make(map[string]*YamlData), cfg.Files...)
	if err != nil {
		return "", &errors.YamllError{Message: fmt.Sprintf("fetching dependency tree errored with: '%v'", err)}
	}

	var importData string

	finalData, err := cfg.MergeData(importData, dependencyRoutes)
	if err != nil {
		return "", err
	}

	if cfg.Effective {
		effectiveMergedYaml, err := finalData.EffectiveMerge()
		if err != nil {
			return "", err
		}

		finalData = effectiveMergedYaml
	}

	return finalData, nil
}

// New returns new instance of Config with passed parameters.
func New(effective bool, logLevel, limiter string, paths ...string) *Config {
	dependencies := make([]*Dependency, 0)

	for _, path := range paths {
		dependency := &Dependency{Path: path}
		dependency.identifyType()
		dependencies = append(dependencies, dependency)
	}

	return &Config{Files: dependencies, Limiter: limiter, LogLevel: logLevel, Effective: effective}
}
