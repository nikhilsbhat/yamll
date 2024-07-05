package yamll

import (
	"fmt"
	"log/slog"

	"github.com/fatih/color"
	"github.com/nikhilsbhat/yamll/pkg/errors"
	"github.com/thoas/go-funk"
)

type YamlTree struct {
	Value       string
	Left, Right *YamlTree
}

// mergeData combines the YAML file data according to the hierarchy.
func (cfg *Config) mergeData(src string, routes YamlRoutes) (Yaml, error) {
	for file, fileData := range routes {
		if !fileData.Root {
			continue
		}

		out, err := cfg.merge(src, routes, file)
		if err != nil {
			return "", err
		}

		src = out + fmt.Sprintf("\n%s\n# Source: %s\n%s\n", cfg.Limiter, file, fileData.DataRaw)

		cfg.log.Debug("root file was imported successfully", slog.String("file", file))
	}

	return Yaml(src), nil
}

// merge actually merges the data when invoked with correct parameters.
func (cfg *Config) merge(src string, routes YamlRoutes, file string) (string, error) {
	for _, dependency := range routes[file].Dependency {
		if err := routes.checkInterDependency(file, dependency.Path); err != nil {
			return "", err
		}

		cfg.log.Debug("importing YAML file", slog.String("path", dependency.Path))

		if routes[dependency.Path].Merged {
			cfg.log.Warn("file already imported hence skipping", slog.String("file", dependency.Path))

			continue
		}

		out, err := cfg.merge(src, routes, dependency.Path)
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

// checkInterDependency verifies for deadlock dependencies and raises an error if two YAML files import each other.
func (yamlRoutes YamlRoutes) checkInterDependency(file, dependency string) error {
	if funk.Contains(yamlRoutes[file].Dependency, func(dep *Dependency) bool {
		return dep.Path == dependency
	}) && funk.Contains(yamlRoutes[dependency].Dependency, func(dep *Dependency) bool {
		return dep.Path == file
	}) {
		return &errors.YamllError{Message: fmt.Sprintf("import cycles not allowed '%s' '%s'", file, dependency)}
	}

	return nil
}

// PrintDependencyTree recursively prints the dependency tree.
func (yamlRoutes YamlRoutes) PrintDependencyTree(name string, prefix string, isTail, noColor bool) {
	color.NoColor = noColor

	if data, exists := yamlRoutes[name]; exists {
		connector := color.GreenString("├── ")
		if isTail {
			connector = color.GreenString("└── ")
		}

		fmt.Printf("%s%s%s\n", prefix, connector, color.MagentaString(name))

		newPrefix := prefix
		if isTail {
			newPrefix += "    "
		} else {
			newPrefix += color.GreenString("│   ")
		}

		for i, dep := range data.Dependency {
			yamlRoutes.PrintDependencyTree(dep.Path, newPrefix, i == len(data.Dependency)-1, noColor)
		}
	}
}
