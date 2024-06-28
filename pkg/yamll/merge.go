package yamll

import (
	"fmt"
	"log/slog"

	"github.com/nikhilsbhat/yamll/pkg/errors"
	"github.com/thoas/go-funk"
)

// MergeData combines the YAML file data according to the hierarchy.
func (cfg *Config) MergeData(src string, routes YamlRoutes) (Yaml, error) {
	for file, fileData := range routes {
		if !fileData.Root {
			continue
		}

		out, err := cfg.Merge(src, routes, file)
		if err != nil {
			return "", err
		}

		src = out + fmt.Sprintf("\n%s\n# Source: %s\n%s\n", cfg.Limiter, file, fileData.DataRaw)

		cfg.log.Debug("root file was imported successfully", slog.String("file", file))
	}

	return Yaml(src), nil
}

// Merge actually merges the data when invoked with correct parameters.
func (cfg *Config) Merge(src string, routes YamlRoutes, file string) (string, error) {
	for _, dependency := range routes[file].Dependency {
		if err := routes.CheckInterDependency(file, dependency.Path); err != nil {
			return "", err
		}

		cfg.log.Debug("importing YAML file", slog.String("path", dependency.Path))

		if routes[dependency.Path].Merged {
			cfg.log.Warn("file already imported hence skipping", slog.String("file", dependency.Path))

			continue
		}

		out, err := cfg.Merge(src, routes, dependency.Path)
		if err != nil {
			return "", err
		}

		src = out
	}

	if !routes[file].Merged && !routes[file].Root {
		src = fmt.Sprintf("%s\n%s\n# Source: %s\n%s", src, cfg.Limiter, routes[file].File, routes[file].DataRaw)

		routes[file].Merged = true

		cfg.log.Debug("file was imported successfully", slog.String("file", file))
	}

	return src, nil
}

// CheckInterDependency verifies for deadlock dependencies and raises an error if two YAML files import each other.
func (yamlRoutes YamlRoutes) CheckInterDependency(file, dependency string) error {
	if funk.Contains(yamlRoutes[file].Dependency, func(dep *Dependency) bool {
		return dep.Path == dependency
	}) && funk.Contains(yamlRoutes[dependency].Dependency, func(dep *Dependency) bool {
		return dep.Path == file
	}) {
		return &errors.YamllError{Message: fmt.Sprintf("import cycles not allowed '%s' '%s'", file, dependency)}
	}

	return nil
}
