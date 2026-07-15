package yamll

import (
	stdErrors "errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	yamllerrors "github.com/nikhilsbhat/yamll/pkg/errors"
	yamlv3 "gopkg.in/yaml.v3"
)

type LintReport struct {
	Issues []LintIssue
}

func (report LintReport) HasErrors() bool {
	for _, issue := range report.Issues {
		if issueSeverity(issue) == LintSeverityError {
			return true
		}
	}

	return false
}

type LintIssue struct {
	Severity string
	Code     string
	File     string
	Message  string
}

const (
	LintSeverityError         = "error"
	LintSeverityWarning       = "warning"
	LintDuplicateKeys         = "duplicate-keys"
	LintUnresolvedImports     = "unresolved-imports"
	LintUnusedImports         = "unused-imports"
	LintCircularRefs          = "circular-refs"
	LintInvalidAnchors        = "invalid-anchors"
	LintConflictingMerges     = "conflicting-merges"
	LintDeadImports           = "dead-imports"
	LintOverriddenImports     = "overridden-imports"
	LintDuplicateLibraries    = "duplicate-libraries"
	duplicateLibraryThreshold = 2
)

func (cfg *Config) Lint() (LintReport, error) {
	cfg.Root = false

	routes, unresolvedImportMessage := cfg.resolveLintRoutes()
	if unresolvedImportMessage != "" {
		return LintReport{
			Issues: []LintIssue{{
				Severity: LintSeverityError,
				Code:     LintUnresolvedImports,
				Message:  unresolvedImportMessage,
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
	keyPathsByFile := collectRouteKeyPathsByFile(yamlRoutes)
	rootKeyPaths := collectRootKeyPaths(yamlRoutes, keyPathsByFile)

	issues = append(issues, lintInvalidAnchors(anchorDefs, anchorRefs)...)
	issues = append(issues, lintUnusedImports(yamlRoutes, anchorDefs, anchorRefs)...)
	issues = append(issues, lintConflictingMerges(yamlRoutes)...)
	issues = append(issues, lintDeadImports(yamlRoutes, anchorDefs, rootKeyPaths, keyPathsByFile)...)
	issues = append(issues, lintOverriddenImports(yamlRoutes, rootKeyPaths, keyPathsByFile)...)
	issues = append(issues, lintDuplicateLibraries(yamlRoutes)...)

	sort.SliceStable(issues, func(index, jIndex int) bool {
		if issues[index].Severity != issues[jIndex].Severity {
			return issues[index].Severity < issues[jIndex].Severity
		}

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

func (cfg *Config) resolveLintRoutes() (map[string]*YamlData, string) {
	routes, err := cfg.ResolveDependencies(make(map[string]*YamlData), cfg.Files...)
	if err != nil {
		return nil, err.Error()
	}

	return routes, ""
}

func issueSeverity(issue LintIssue) string {
	if issue.Severity == "" {
		return LintSeverityError
	}

	return issue.Severity
}

func lintCircularRefs(routes YamlRoutes) []LintIssue {
	issues := make([]LintIssue, 0)

	visiting := make(map[string]struct{})
	visited := make(map[string]struct{})

	var dfs func(file string, stack []string)

	dfs = func(file string, stack []string) {
		if _, exists := visiting[file]; exists {
			// cycle
			issues = append(issues, LintIssue{
				Severity: LintSeverityError,
				Code:     LintCircularRefs,
				File:     file,
				Message:  fmt.Sprintf("import cycle detected: %s -> %s", strings.Join(stack, " -> "), file),
			})

			return
		}

		if _, exists := visited[file]; exists {
			return
		}

		visiting[file] = struct{}{}

		defer delete(visiting, file)

		route := routes[file]
		if route != nil {
			for _, dep := range route.Dependency {
				if dep == nil {
					continue
				}

				dfs(dep.Path, append(stack, file))
			}
		}

		visited[file] = struct{}{}
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
				Severity: LintSeverityError,
				Code:     LintInvalidAnchors,
				File:     file,
				Message:  fmt.Sprintf("unknown anchor reference '*%s'", refName),
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
				Severity: LintSeverityWarning,
				Code:     LintUnusedImports,
				File:     route.File,
				Message:  "imported file appears unused (no anchors referenced)",
			})
		}
	}

	return issues
}

func lintDuplicateKeys(routes YamlRoutes) []LintIssue {
	issues := make([]LintIssue, 0, len(routes))

	for _, file := range routes.OrderedFiles() {
		route := routes[file]
		if route == nil {
			continue
		}

		for _, src := range route.SourceFile {
			// Replace *alias with a scalar so strict parsing doesn't fail due to cross-file anchors.
			sanitized := aliasRefPattern.ReplaceAllString(src.Data, `${1}"__yamll_lint_alias_${2}"`)

			var node yamlv3.Node
			if err := yamlv3.Unmarshal([]byte(sanitized), &node); err != nil {
				continue
			}

			lintDuplicateKeysInNode(src.Name, &node, &issues)
		}
	}

	return issues
}

func lintDuplicateKeysInNode(file string, node *yamlv3.Node, issues *[]LintIssue) {
	if node == nil {
		return
	}

	if node.Kind == yamlv3.MappingNode {
		seen := make(map[string]int)

		for index := 0; index < len(node.Content)-1; index += 2 {
			keyNode := node.Content[index]
			valueNode := node.Content[index+1]
			key := keyNode.Value

			if key == "<<" {
				lintDuplicateKeysInNode(file, valueNode, issues)

				continue
			}

			if firstLine, exists := seen[key]; exists {
				*issues = append(*issues, LintIssue{
					Severity: LintSeverityError,
					Code:     LintDuplicateKeys,
					File:     file,
					Message:  fmt.Sprintf("duplicate key %q at line %d; first defined at line %d", key, keyNode.Line, firstLine),
				})
			} else {
				seen[key] = keyNode.Line
			}

			lintDuplicateKeysInNode(file, valueNode, issues)
		}

		return
	}

	for _, child := range node.Content {
		lintDuplicateKeysInNode(file, child, issues)
	}
}

func lintConflictingMerges(routes YamlRoutes) []LintIssue {
	issues := make([]LintIssue, 0)

	anchorRefData := dedupeAnchorReferences(routes.getRawData())

	anchorKinds, err := collectAnchorKinds(anchorRefData)
	if err != nil {
		return []LintIssue{{
			Severity: LintSeverityError,
			Code:     LintConflictingMerges,
			Message:  err.Error(),
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
						Severity: LintSeverityError,
						Code:     LintConflictingMerges,
						File:     src.Name,
						Message:  yerr.Message,
					})
				} else {
					issues = append(issues, LintIssue{
						Severity: LintSeverityError,
						Code:     LintConflictingMerges,
						File:     src.Name,
						Message:  err.Error(),
					})
				}
			}
		}
	}

	return issues
}

func lintDeadImports(
	routes YamlRoutes,
	defs map[string]map[string]struct{},
	rootKeyPaths map[string]map[string]struct{},
	keyPathsByFile map[string]map[string]struct{},
) []LintIssue {
	issues := make([]LintIssue, 0)

	for _, file := range routes.OrderedFiles() {
		route := routes[file]
		if route == nil || route.Root {
			continue
		}

		keyPaths := keyPathsByFile[route.File]
		if len(keyPaths) == 0 {
			issues = append(issues, LintIssue{
				Severity: LintSeverityWarning,
				Code:     LintDeadImports,
				File:     route.File,
				Message:  fmt.Sprintf("%s imported but contributes nothing", route.File),
			})

			continue
		}

		if allKeysOverridden(keyPaths, rootKeyPaths) && len(defs) == 0 {
			issues = append(issues, LintIssue{
				Severity: LintSeverityWarning,
				Code:     LintDeadImports,
				File:     route.File,
				Message:  fmt.Sprintf("%s imported but contributes nothing", route.File),
			})
		}
	}

	return issues
}

func lintOverriddenImports(
	routes YamlRoutes,
	rootKeyPaths map[string]map[string]struct{},
	keyPathsByFile map[string]map[string]struct{},
) []LintIssue {
	issues := make([]LintIssue, 0)

	for _, file := range routes.OrderedFiles() {
		route := routes[file]
		if route == nil || route.Root {
			continue
		}

		keyPaths := keyPathsByFile[route.File]
		overridden := make([]string, 0)

		for keyPath := range keyPaths {
			if allRootsContain(rootKeyPaths, keyPath) {
				overridden = append(overridden, keyPath)
			}
		}

		if len(overridden) == 0 {
			continue
		}

		sort.Strings(overridden)
		issues = append(issues, LintIssue{
			Severity: LintSeverityWarning,
			Code:     LintOverriddenImports,
			File:     route.File,
			Message:  fmt.Sprintf("%s overridden in every environment: %s", route.File, strings.Join(overridden, ", ")),
		})
	}

	return issues
}

func lintDuplicateLibraries(routes YamlRoutes) []LintIssue {
	issues := make([]LintIssue, 0)

	contentSources := make(map[string][]string)

	for _, file := range routes.OrderedFiles() {
		route := routes[file]
		if route == nil {
			continue
		}

		for _, src := range route.SourceFile {
			if src.Meta.SHA256 == "" {
				continue
			}

			contentSources[src.Meta.SHA256] = append(contentSources[src.Meta.SHA256], src.Name)
		}
	}

	for hash, sources := range contentSources {
		if len(sources) < duplicateLibraryThreshold {
			continue
		}

		sort.Strings(sources)
		issues = append(issues, LintIssue{
			Severity: LintSeverityWarning,
			Code:     LintDuplicateLibraries,
			File:     strings.Join(sources, ", "),
			Message:  fmt.Sprintf("same content imported from multiple locations (sha256=%s)", hash),
		})
	}

	return issues
}

func collectRootKeyPaths(routes YamlRoutes, keyPathsByFile map[string]map[string]struct{}) map[string]map[string]struct{} {
	roots := make(map[string]map[string]struct{})

	for _, file := range routes.OrderedFiles() {
		route := routes[file]
		if route == nil || !route.Root {
			continue
		}

		roots[route.File] = keyPathsByFile[route.File]
	}

	return roots
}

func collectRouteKeyPathsByFile(routes YamlRoutes) map[string]map[string]struct{} {
	keyPaths := make(map[string]map[string]struct{}, len(routes))

	for _, file := range routes.OrderedFiles() {
		route := routes[file]
		if route == nil {
			continue
		}

		keyPaths[route.File] = collectRouteKeyPaths(route)
	}

	return keyPaths
}

func collectRouteKeyPaths(route *YamlData) map[string]struct{} {
	keyPaths := make(map[string]struct{})

	if route == nil {
		return keyPaths
	}

	for _, src := range route.SourceFile {
		keyPathsFromYAML(src.Data, "", keyPaths)
	}

	return keyPaths
}

func keyPathsFromYAML(data, prefix string, out map[string]struct{}) {
	var node yamlv3.Node
	if err := yamlv3.Unmarshal([]byte(data), &node); err != nil || len(node.Content) == 0 {
		return
	}

	collectKeyPathsFromNode(node.Content[0], prefix, out)
}

func collectKeyPathsFromNode(node *yamlv3.Node, prefix string, out map[string]struct{}) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yamlv3.DocumentNode:
		for _, child := range node.Content {
			collectKeyPathsFromNode(child, prefix, out)
		}
	case yamlv3.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			if keyNode == nil {
				continue
			}

			key := keyNode.Value
			if key == "" {
				continue
			}

			path := key
			if prefix != "" {
				path = prefix + "." + key
			}

			out[path] = struct{}{}
			collectKeyPathsFromNode(valueNode, path, out)
		}
	case yamlv3.SequenceNode:
		for _, child := range node.Content {
			collectKeyPathsFromNode(child, prefix, out)
		}
	case yamlv3.ScalarNode, yamlv3.AliasNode:
		return
	default:
		return
	}
}

func allRootsContain(rootKeyPaths map[string]map[string]struct{}, keyPath string) bool {
	if len(rootKeyPaths) == 0 {
		return false
	}

	for _, rootPaths := range rootKeyPaths {
		if _, ok := rootPaths[keyPath]; !ok {
			return false
		}
	}

	return true
}

func allKeysOverridden(keys map[string]struct{}, rootKeyPaths map[string]map[string]struct{}) bool {
	if len(keys) == 0 || len(rootKeyPaths) == 0 {
		return false
	}

	for keyPath := range keys {
		if !allRootsContain(rootKeyPaths, keyPath) {
			return false
		}
	}

	return true
}
