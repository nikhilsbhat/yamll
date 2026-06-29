package yamll

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/nikhilsbhat/yamll/pkg/errors"
)

// YamlData holds information of yaml file and its dependency tree.
type YamlData struct {
	Root       bool          `json:"root,omitempty" yaml:"root,omitempty"`
	Merged     bool          `json:"merged,omitempty" yaml:"merged,omitempty"`
	Index      int           `json:"index,omitempty" yaml:"index,omitempty"`
	File       string        `json:"file,omitempty" yaml:"file,omitempty"`
	DataRaw    string        `json:"data_raw,omitempty" yaml:"data_raw,omitempty"`
	Dependency []*Dependency `json:"dependency,omitempty" yaml:"dependency,omitempty"`
	SourceFile []File        `json:"-" yaml:"-"`
}

// Config holds the information of yaml files to be parsed.
type Config struct {
	Root     bool          `json:"root,omitempty" yaml:"root,omitempty"`
	Merge    bool          `json:"effective,omitempty" yaml:"effective,omitempty"`
	Split    bool          `json:"split,omitempty" yaml:"split,omitempty"`
	Limiter  string        `json:"limiter,omitempty" yaml:"limiter,omitempty"`
	LogLevel string        `json:"log_level,omitempty" yaml:"log_level,omitempty"`
	Files    []*Dependency `json:"files,omitempty" yaml:"files,omitempty"`
	LockFile string        `json:"lock_file,omitempty" yaml:"lock_file,omitempty"`
	NoLock   bool          `json:"no_lock,omitempty" yaml:"no_lock,omitempty"`
	Profile  bool          `json:"profile,omitempty" yaml:"profile,omitempty"`
	log      *slog.Logger
	profile  *BuildProfile
}

// YamlRoutes holds a map of YamlData, representing a dependency tree.
type YamlRoutes map[string]*YamlData

// Yaml is a string representation of YAML content.
type Yaml string

// Yaml resolves shared YAML imports into one coherent output, while preserving where each piece came from.
// These imports work in a manner similar to importing libraries in a programming language.
// It searches for the imports defined in any of the following (comments that start with ##++ in your YAML definition).
// Supports importing from various sources including local files, URLs, Git, and OCI artifacts.
// Sample imports look like:
//
//	##++internal/fixtures/base2.yaml
//	##++https://run.mocky.io/v3/92e08b25-dd1f-4dd0-bc55-9649b5b896c9
//	##++git+https://github.com/nikhilsbhat/yamll@main?path=internal/fixtures/base.yaml
//	##++git+ssh://git@github.com:nikhilsbhat/yamll@main?path=internal/fixtures/base.yaml
//	##++git+ssh://git@github.com:nikhilsbhat/yamll@v0.2.5?path=internal/fixtures/base.yaml
//	##++oci://ghcr.io/company/platform-config:v1
//	##++https://test.com/test.yaml;{"user_name":"${username}","password":"${pass}","ca_content":"${ca_content}"}
//
// The parameters necessary for authenticating the remote server in URL/GIT/OCI based imports should be defined as shown in the example above.
// All supported parameters can found under Auth.
//
// Authentication parameters, which cannot be directly specified in imports for security reasons, can be replaced with environment variables.
// To use this feature, define the parameter exposed as an environment variable as $VARIABLE_NAME, as shown in the last example.
//
// Breakdown of git repo based import:
// http based url: ##++git+https://github.com/<org_name>/<repo_name>@<branch/tag>?path=<path/to/file.yaml> ex: ##++git+https://github.com/nikhilsbhat/yamll@main?path=internal/fixtures/base.yaml.
// ssh based url ##++git+ssh://git@github.com:<org_name>/<repo_name>@<branch/tag>?path=<path/to/file.yaml> ex: ##++git+ssh://git@github.com:nikhilsbhat/yamll@main?path=internal/fixtures/base.yaml.
// OCI based url: ##++oci://ghcr.io/<org_name>/<artifact>:<tag> ex: ##++oci://ghcr.io/company/platform-config:v1.
//
//nolint:lll
func (cfg *Config) Yaml() (Yaml, error) {
	cfg.Root = false

	dependencyRoutes, err := cfg.ResolveDependencies(make(map[string]*YamlData), cfg.Files...)
	if err != nil {
		return "", &errors.YamllError{Message: fmt.Sprintf("fetching dependency tree errored with: '%v'", err)}
	}

	var importData string

	finalData, err := cfg.mergeData(importData, dependencyRoutes)
	if err != nil {
		return "", err
	}

	if cfg.Merge && !cfg.Split {
		effectiveMergedYaml, err := finalData.EffectiveMerge()
		if err != nil {
			return "", err
		}

		finalData = effectiveMergedYaml
	}

	return finalData, nil
}

// YamlTree constructs a dependency tree and displays it in a format similar to the Linux tree utility.
func (cfg *Config) YamlTree(color bool, showPatternFiles bool) error {
	output, err := cfg.Tree(TreeOutputText, color, showPatternFiles)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(os.Stdout, output)

	return err
}

func (cfg *Config) Tree(outputFormat string, noColor, showPatternFiles bool) (string, error) {
	cfg.Root = false

	dependencyRoutes, err := cfg.ResolveDependencies(make(map[string]*YamlData), cfg.Files...)
	if err != nil {
		return "", &errors.YamllError{Message: fmt.Sprintf("fetching dependency tree errored with: '%v'", err)}
	}

	rootFile := cfg.Files[0].Path

	cfg.log.Debug("identified root file", slog.Any("file", rootFile), slog.String("output", normalizeTreeOutputFormat(outputFormat)))

	return YamlRoutes(dependencyRoutes).RenderDependencyTree(rootFile, outputFormat, noColor, showPatternFiles)
}

// YamlBuild builds YAML by substituting all anchors and aliases defined in sub-YAML files defined as libraries.
func (cfg *Config) YamlBuild() (Yaml, error) {
	cfg.Root = false
	if cfg.Profile {
		cfg.profile = &BuildProfile{}
		cfg.profile.begin()
	}

	resolveStart := time.Now()

	dependencyRoutes, err := cfg.ResolveDependencies(make(map[string]*YamlData), cfg.Files...)
	if err != nil {
		return "", &errors.YamllError{Message: fmt.Sprintf("fetching dependency tree errored with: '%v'", err)}
	}

	if cfg.Profile && cfg.profile != nil {
		cfg.profile.addImportResolution(time.Since(resolveStart))
	}

	mergeStart := time.Now()

	merged, err := YamlRoutes(dependencyRoutes).Build()
	if err != nil {
		return "", err
	}

	if cfg.Profile && cfg.profile != nil {
		cfg.profile.addMerge(time.Since(mergeStart))
	}

	return merged, nil
}

func (cfg *Config) ProfileReport() string {
	if cfg.profile == nil {
		return ""
	}

	return cfg.profile.String()
}

func (cfg *Config) RecordValidationTiming(d time.Duration) {
	if cfg.profile != nil {
		cfg.profile.addValidation(d)
	}
}

// New returns new instance of Config with passed parameters.
func New(effective bool, logLevel, limiter string, paths ...string) *Config {
	dependencies := make([]*Dependency, 0, len(paths))

	for _, path := range paths {
		dependency := &Dependency{Path: path}
		dependency.IdentifyType()
		dependencies = append(dependencies, dependency)
	}

	return &Config{
		Files:    dependencies,
		Limiter:  limiter,
		LogLevel: logLevel,
		Merge:    effective,
		LockFile: "yamll.lock",
	}
}
