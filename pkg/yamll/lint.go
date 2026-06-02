package yamll

import (
	stdErrors "errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
	yamllerrors "github.com/nikhilsbhat/yamll/pkg/errors"
)

type LintReport struct {
	Issues []LintIssue
}

type LintIssue struct {
	Code    string
	File    string
	Message string
}

const (
	LintDuplicateKeys     = "duplicate-keys"
	LintUnresolvedImports = "unresolved-imports"
	LintUnusedImports     = "unused-imports"
	LintCircularRefs      = "circular-refs"
	LintInvalidAnchors    = "invalid-anchors"
	LintConflictingMerges = "conflicting-merges"
)

func (cfg *Config) Lint() (LintReport, error) {
	cfg.Root = false

	routes, unresolvedImportMessage := func() (map[string]*YamlData, string) {
		resolvedRoutes, err := cfg.ResolveDependencies(make(map[string]*YamlData), cfg.Files...)
		if err != nil {
			return nil, err.Error()
		}

		return resolvedRoutes, ""
	}()

	if unresolvedImportMessage != "" {
		return LintReport{
			Issues: []LintIssue{{
				Code:    LintUnresolvedImports,
				Message: unresolvedImportMessage,
			}},
		}, nil
	}

	yamlRoutes := YamlRoutes(routes)

	const defaultAddon = 2

	issues := make([]LintIssue, 0, len(yamlRoutes)*defaultAddon)

	issues = append(issues, lintCircularRefs(yamlRoutes)...)
	issues = append(issues, lintDuplicateKeys(yamlRoutes)...)

	anchorDefs := collectAnchorDefs(yamlRoutes)
	anchorRefs := collectAnchorRefs(yamlRoutes)

	issues = append(issues, lintInvalidAnchors(anchorDefs, anchorRefs)...)
	issues = append(issues, lintUnusedImports(yamlRoutes, anchorDefs, anchorRefs)...)
	issues = append(issues, lintConflictingMerges(yamlRoutes)...)

	sort.SliceStable(issues, func(index, jIndex int) bool {
		if issues[index].Code != issues[jIndex].Code {
			return issues[index].Code < issues[jIndex].Code
		}

		if issues[index].File != issues[jIndex].File {
			return issues[index].File < issues[jIndex].File
		}

		return issues[index].Message < issues[jIndex].Message
	})

	return LintReport{Issues: issues}, nil
}

func lintCircularRefs(routes YamlRoutes) []LintIssue {
	issues := make([]LintIssue, 0)

	visiting := make(map[string]bool)
	visited := make(map[string]bool)

	var dfs func(file string, stack []string)

	dfs = func(file string, stack []string) {
		if visiting[file] {
			// cycle
			issues = append(issues, LintIssue{
				Code:    LintCircularRefs,
				File:    file,
				Message: fmt.Sprintf("import cycle detected: %s -> %s", strings.Join(stack, " -> "), file),
			})

			return
		}

		if visited[file] {
			return
		}

		visiting[file] = true

		defer func() { visiting[file] = false }()

		route := routes[file]
		if route != nil {
			for _, dep := range route.Dependency {
				if dep == nil {
					continue
				}

				dfs(dep.Path, append(stack, file))
			}
		}

		visited[file] = true
	}

	for _, file := range routes.OrderedFiles() {
		dfs(file, nil)
	}

	return issues
}

var aliasRefPattern = regexp.MustCompile(`(^|[\s\[{,])\*([A-Za-z0-9_-]+)`)

func collectAnchorRefs(routes YamlRoutes) map[string]map[string]struct{} {
	refs := make(map[string]map[string]struct{})

	for _, file := range routes.OrderedFiles() {
		route := routes[file]
		if route == nil {
			continue
		}

		for _, src := range route.SourceFile {
			matches := aliasRefPattern.FindAllStringSubmatch(src.Data, -1)
			if len(matches) == 0 {
				continue
			}

			for _, m := range matches {
				name := m[2]
				if name == "" {
					continue
				}

				if _, ok := refs[name]; !ok {
					refs[name] = make(map[string]struct{})
				}

				refs[name][src.Name] = struct{}{}
			}
		}
	}

	return refs
}

var anchorDefPattern = regexp.MustCompile(`&([A-Za-z0-9_-]+)`)

func collectAnchorDefs(routes YamlRoutes) map[string]map[string]struct{} {
	defs := make(map[string]map[string]struct{})

	for _, file := range routes.OrderedFiles() {
		route := routes[file]
		if route == nil {
			continue
		}

		for _, src := range route.SourceFile {
			matches := anchorDefPattern.FindAllStringSubmatch(src.Data, -1)
			if len(matches) == 0 {
				continue
			}

			for _, m := range matches {
				name := m[1]
				if name == "" {
					continue
				}

				if _, ok := defs[name]; !ok {
					defs[name] = make(map[string]struct{})
				}

				defs[name][src.Name] = struct{}{}
			}
		}
	}

	return defs
}

func lintInvalidAnchors(defs map[string]map[string]struct{}, refs map[string]map[string]struct{}) []LintIssue {
	issues := make([]LintIssue, 0)

	for refName, files := range refs {
		if _, ok := defs[refName]; ok {
			continue
		}

		for file := range files {
			issues = append(issues, LintIssue{
				Code:    LintInvalidAnchors,
				File:    file,
				Message: fmt.Sprintf("unknown anchor reference '*%s'", refName),
			})
		}
	}

	return issues
}

func lintUnusedImports(routes YamlRoutes, defs map[string]map[string]struct{}, refs map[string]map[string]struct{}) []LintIssue {
	issues := make([]LintIssue, 0)

	// Build: file -> anchors defined
	definedByFile := make(map[string]map[string]struct{})

	for anchorName, files := range defs {
		for file := range files {
			fileMap := definedByFile[file]
			if fileMap == nil {
				fileMap = make(map[string]struct{})
				definedByFile[file] = fileMap
			}

			fileMap[anchorName] = struct{}{}
		}
	}

	referencedAnchor := make(map[string]struct{}, len(refs))
	for anchor := range refs {
		referencedAnchor[anchor] = struct{}{}
	}

	// For each non-root route, if none of its anchors are referenced anywhere, flag as unused.
	for _, file := range routes.OrderedFiles() {
		route := routes[file]
		if route == nil || route.Root {
			continue
		}

		sourceNames := make(map[string]struct{})
		if len(route.SourceFile) == 0 {
			sourceNames[route.File] = struct{}{}
		} else {
			for _, src := range route.SourceFile {
				sourceNames[src.Name] = struct{}{}
			}
		}

		used := false

		for srcName := range sourceNames {
			for anchor := range definedByFile[srcName] {
				if _, ok := referencedAnchor[anchor]; ok {
					used = true

					break
				}
			}

			if used {
				break
			}
		}

		if !used {
			issues = append(issues, LintIssue{
				Code:    LintUnusedImports,
				File:    route.File,
				Message: "imported file appears unused (no anchors referenced)",
			})
		}
	}

	return issues
}

func lintDuplicateKeys(routes YamlRoutes) []LintIssue {
	issues := make([]LintIssue, 0)

	for _, file := range routes.OrderedFiles() {
		route := routes[file]
		if route == nil {
			continue
		}

		for _, src := range route.SourceFile {
			// Replace *alias with a scalar so strict parsing doesn't fail due to cross-file anchors.
			sanitized := aliasRefPattern.ReplaceAllString(src.Data, `${1}"__yamll_lint_alias_${2}"`)

			var out yaml.MapSlice
			if err := yaml.UnmarshalWithOptions([]byte(sanitized), &out, yaml.UseOrderedMap(), yaml.DisallowDuplicateKey(), yaml.Strict()); err != nil {
				msg := err.Error()
				if strings.Contains(strings.ToLower(msg), "duplicate") && strings.Contains(strings.ToLower(msg), "key") {
					issues = append(issues, LintIssue{
						Code:    LintDuplicateKeys,
						File:    src.Name,
						Message: msg,
					})
				}
			}
		}
	}

	return issues
}

func lintConflictingMerges(routes YamlRoutes) []LintIssue {
	issues := make([]LintIssue, 0)

	anchorRefData := dedupeAnchorReferences(routes.getRawData())

	anchorKinds, err := collectAnchorKinds(anchorRefData)
	if err != nil {
		return []LintIssue{{
			Code:    LintConflictingMerges,
			Message: err.Error(),
		}}
	}

	for _, file := range routes.OrderedFiles() {
		route := routes[file]
		if route == nil {
			continue
		}

		for _, src := range route.SourceFile {
			if err = validateMergeAliases(src.Data, anchorKinds); err != nil {
				var yerr *yamllerrors.YamllError

				if ok := stdErrors.As(err, &yerr); ok {
					issues = append(issues, LintIssue{
						Code:    LintConflictingMerges,
						File:    src.Name,
						Message: yerr.Message,
					})
				} else {
					issues = append(issues, LintIssue{
						Code:    LintConflictingMerges,
						File:    src.Name,
						Message: err.Error(),
					})
				}
			}
		}
	}

	return issues
}
