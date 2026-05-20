package yamll

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/fatih/color"
	"github.com/nikhilsbhat/yamll/pkg/errors"
	"github.com/thoas/go-funk"
)

// YamlTree holds the information of defined yaml dependencies.
type YamlTree struct {
	Value       string
	Left, Right *YamlTree
}

// mergeData combines the YAML file data according to the hierarchy.
func (cfg *Config) mergeData(src string, routes YamlRoutes) (Yaml, error) {
	for _, file := range cfg.rootFiles(routes) {
		fileData := routes[file]

		out, err := cfg.merge(src, routes, file, make(map[string]bool))
		if err != nil {
			return "", err
		}

		src = out + fmt.Sprintf("\n%s\n# Source: %s\n%s\n", cfg.Limiter, file, fileData.DataRaw)

		cfg.log.Debug("root file was imported successfully", slog.String("file", file))
	}

	return Yaml(src), nil
}

// merge actually merges the data when invoked with correct parameters.
func (cfg *Config) merge(src string, routes YamlRoutes, file string, visiting map[string]bool) (string, error) {
	route, exists := routes[file]
	if !exists {
		return "", &errors.YamllError{Message: fmt.Sprintf("dependency route missing for '%s'", file)}
	}

	if visiting[file] {
		return "", &errors.YamllError{Message: fmt.Sprintf("import cycle detected at '%s'", file)}
	}

	visiting[file] = true
	defer delete(visiting, file)

	for _, dependency := range routes[file].Dependency {
		if _, exists := routes[dependency.Path]; !exists {
			return "", &errors.YamllError{Message: fmt.Sprintf("dependency route missing for '%s'", dependency.Path)}
		}

		cfg.log.Debug("importing YAML file", slog.String("path", dependency.Path))

		if routes[dependency.Path].Merged {
			cfg.log.Warn("file already imported hence skipping", slog.String("file", dependency.Path))

			continue
		}

		out, err := cfg.merge(src, routes, dependency.Path, visiting)
		if err != nil {
			return "", err
		}

		src = out
	}

	if !route.Merged && !route.Root {
		src = fmt.Sprintf("%s\n%s\n# Source: %s\n%s", src, cfg.Limiter, route.File, route.DataRaw)

		route.Merged = true

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

func (cfg *Config) rootFiles(routes YamlRoutes) []string {
	var files []string

	seen := make(map[string]struct{}, len(cfg.Files))

	for _, dependency := range cfg.Files {
		if route, exists := routes[dependency.Path]; exists && route.Root {
			files = append(files, dependency.Path)
			seen[dependency.Path] = struct{}{}
		}
	}

	for file, route := range routes {
		if !route.Root {
			continue
		}

		if _, exists := seen[file]; exists {
			continue
		}

		files = append(files, file)
	}

	sort.SliceStable(files, func(i, j int) bool {
		return routeLess(routes, files[i], files[j])
	})

	return files
}

func (yamlRoutes YamlRoutes) OrderedFiles() []string {
	var (
		rootFiles     []string
		leftoverFiles []string
	)

	for file, route := range yamlRoutes {
		if route.Root {
			rootFiles = append(rootFiles, file)
		}
	}

	sort.SliceStable(rootFiles, func(i, j int) bool {
		return routeLess(yamlRoutes, rootFiles[i], rootFiles[j])
	})

	seen := make(map[string]struct{}, len(yamlRoutes))
	visiting := make(map[string]struct{}, len(yamlRoutes))
	orderedFiles := make([]string, 0, len(yamlRoutes))

	for _, file := range rootFiles {
		yamlRoutes.appendOrderedFile(file, seen, visiting, &orderedFiles)
	}

	for file := range yamlRoutes {
		if _, exists := seen[file]; exists {
			continue
		}

		leftoverFiles = append(leftoverFiles, file)
	}

	sort.SliceStable(leftoverFiles, func(i, j int) bool {
		return routeLess(yamlRoutes, leftoverFiles[i], leftoverFiles[j])
	})

	orderedFiles = append(orderedFiles, leftoverFiles...)

	return orderedFiles
}

func (yamlRoutes YamlRoutes) appendOrderedFile(file string, seen map[string]struct{}, visiting map[string]struct{}, orderedFiles *[]string) {
	if _, exists := seen[file]; exists {
		return
	}

	if _, exists := visiting[file]; exists {
		return
	}

	route, exists := yamlRoutes[file]
	if !exists {
		return
	}

	visiting[file] = struct{}{}

	defer delete(visiting, file)

	for _, dependency := range route.Dependency {
		yamlRoutes.appendOrderedFile(dependency.Path, seen, visiting, orderedFiles)
	}

	seen[file] = struct{}{}

	*orderedFiles = append(*orderedFiles, file)
}

func routeLess(routes YamlRoutes, left, right string) bool {
	leftRoute := routes[left]
	rightRoute := routes[right]

	switch {
	case leftRoute == nil || rightRoute == nil:
		return left < right
	case leftRoute.Root != rightRoute.Root:
		return leftRoute.Root
	case leftRoute.Index != rightRoute.Index:
		return leftRoute.Index < rightRoute.Index
	default:
		return leftRoute.File < rightRoute.File
	}
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
