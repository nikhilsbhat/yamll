package yamll

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nikhilsbhat/yamll/pkg/errors"
)

type ImpactReport struct {
	Target   string
	Affected []string
	Total    int
}

func (cfg *Config) Impact(target string) (ImpactReport, error) {
	cfg.Root = false

	routes, err := cfg.ResolveDependencies(make(map[string]*YamlData), cfg.Files...)
	if err != nil {
		return ImpactReport{}, &errors.YamllError{Message: fmt.Sprintf("fetching dependency tree errored with: '%v'", err)}
	}

	yamlRoutes := YamlRoutes(routes)
	if _, exists := yamlRoutes[target]; !exists {
		return ImpactReport{}, &errors.YamllError{Message: fmt.Sprintf("target file '%s' not found in dependency tree", target)}
	}

	reverseDeps := buildReverseDependencies(yamlRoutes)
	affected := collectImpactedFiles(target, routes, reverseDeps)

	sort.SliceStable(affected, func(i, j int) bool {
		return routeLess(yamlRoutes, affected[i], affected[j])
	})

	return ImpactReport{
		Target:   target,
		Affected: affected,
		Total:    len(affected),
	}, nil
}

func buildReverseDependencies(routes YamlRoutes) map[string]map[string]struct{} {
	reverse := make(map[string]map[string]struct{}, len(routes))

	for file, route := range routes {
		if route == nil {
			continue
		}

		for _, dep := range route.Dependency {
			if dep == nil {
				continue
			}

			if _, ok := reverse[dep.Path]; !ok {
				reverse[dep.Path] = make(map[string]struct{})
			}

			reverse[dep.Path][file] = struct{}{}
		}
	}

	return reverse
}

func collectImpactedFiles(target string, routes map[string]*YamlData, reverse map[string]map[string]struct{}) []string {
	visited := make(map[string]struct{})
	queue := []string{target}
	affected := make([]string, 0, len(reverse))

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for dependent := range reverse[current] {
			if route, ok := routes[dependent]; ok && route != nil && route.Root {
				continue
			}

			if _, seen := visited[dependent]; seen {
				continue
			}

			visited[dependent] = struct{}{}
			affected = append(affected, dependent) //nolint:wsl,wsl_v5
			queue = append(queue, dependent)
		}
	}

	return affected
}

//nolint:wsl,wsl_v5,mnd
func (r ImpactReport) String() string {
	lines := make([]string, 0, len(r.Affected)+2)
	lines = append(lines, "Affected files:")

	for _, file := range r.Affected {
		lines = append(lines, "  "+file)
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Total downstream dependencies: %d", r.Total))

	return strings.Join(lines, "\n") + "\n"
}
