package yamll

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/nikhilsbhat/yamll/pkg/errors"
	yamlv3 "gopkg.in/yaml.v3"
)

const traceAliasPrefix = "__yamll_trace_alias_"

var traceAliasPattern = regexp.MustCompile(`(^|[\s\[{,])\*([A-Za-z0-9_-]+)`)

type TraceResult struct {
	Path   string
	Origin string
	File   string
	Line   int
}

type anchorOrigin struct {
	node *yamlv3.Node
	file string
}

func (cfg *Config) Trace(path string) (TraceResult, error) {
	if len(cfg.Files) == 0 {
		return TraceResult{}, &errors.YamllError{Message: "trace requires a root file"}
	}

	cfg.Root = false

	routes, err := cfg.ResolveDependencies(make(map[string]*YamlData), cfg.Files...)
	if err != nil {
		return TraceResult{}, &errors.YamllError{Message: fmt.Sprintf("fetching dependency tree errored with: '%v'", err)}
	}

	rootFile := cfg.Files[0].Path

	rootRoute, exists := routes[rootFile]
	if !exists {
		return TraceResult{}, &errors.YamllError{Message: fmt.Sprintf("root file '%s' not found in dependency tree", rootFile)}
	}

	parts := strings.Split(strings.TrimSpace(path), ".")
	if len(parts) == 0 || parts[0] == "" {
		return TraceResult{}, &errors.YamllError{Message: "trace path cannot be empty"}
	}

	yamlRoutes := YamlRoutes(routes)

	anchors, err := yamlRoutes.collectAnchors()
	if err != nil {
		return TraceResult{}, err
	}

	for _, sourceFile := range rootRoute.SourceFile {
		node, err := parseYAMLSource(sourceFile.Data)
		if err != nil {
			return TraceResult{}, err
		}

		origin, ok := findTraceOrigin(node, parts, sourceFile.Name, anchors)
		if ok {
			origin.File = displayPath(origin.File)
			origin.Path = path
			origin.Origin = fmt.Sprintf("%s:%d", origin.File, origin.Line)

			return origin, nil
		}
	}

	return TraceResult{}, &errors.YamllError{Message: fmt.Sprintf("path '%s' not found in generated YAML", path)}
}

func (yamlRoutes YamlRoutes) collectAnchors() (map[string]anchorOrigin, error) {
	anchors := make(map[string]anchorOrigin)

	for _, file := range yamlRoutes.OrderedFiles() {
		route := yamlRoutes[file]
		for _, sourceFile := range route.SourceFile {
			node, err := parseYAMLSource(sourceFile.Data)
			if err != nil {
				return nil, err
			}

			collectAnchorsFromNode(node, sourceFile.Name, anchors)
		}
	}

	return anchors, nil
}

func collectAnchorsFromNode(node *yamlv3.Node, file string, anchors map[string]anchorOrigin) {
	if node == nil {
		return
	}

	if node.Anchor != "" {
		if _, exists := anchors[node.Anchor]; !exists {
			anchors[node.Anchor] = anchorOrigin{node: node, file: file}
		}
	}

	for _, child := range node.Content {
		collectAnchorsFromNode(child, file, anchors)
	}
}

func parseYAMLSource(data string) (*yamlv3.Node, error) {
	var node yamlv3.Node

	if err := yamlv3.Unmarshal([]byte(escapeAliasesForTrace(data)), &node); err != nil {
		return nil, &errors.YamllError{Message: fmt.Sprintf("parsing YAML for trace errored with: '%v'", err)}
	}

	if len(node.Content) == 0 {
		return nil, nil
	}

	return node.Content[0], nil
}

func findTraceOrigin(
	node *yamlv3.Node,
	parts []string,
	file string,
	anchors map[string]anchorOrigin,
) (TraceResult, bool) {
	if node == nil {
		return TraceResult{}, false
	}

	if len(parts) == 0 {
		return TraceResult{File: file, Line: node.Line}, true
	}

	if node.Kind == yamlv3.AliasNode {
		anchor, exists := anchors[node.Value]
		if !exists {
			return TraceResult{}, false
		}

		return findTraceOrigin(anchor.node, parts, anchor.file, anchors)
	}

	if node.Kind == yamlv3.ScalarNode && strings.HasPrefix(node.Value, traceAliasPrefix) {
		anchorName := strings.TrimPrefix(node.Value, traceAliasPrefix)

		anchor, exists := anchors[anchorName]
		if !exists {
			return TraceResult{}, false
		}

		return findTraceOrigin(anchor.node, parts, anchor.file, anchors)
	}

	switch node.Kind {
	case yamlv3.MappingNode:
		return findTraceOriginInMapping(node, parts, file, anchors)
	case yamlv3.SequenceNode:
		index, err := strconv.Atoi(parts[0])
		if err != nil || index < 0 || index >= len(node.Content) {
			return TraceResult{}, false
		}

		return findTraceOrigin(node.Content[index], parts[1:], file, anchors)
	case yamlv3.DocumentNode, yamlv3.ScalarNode, yamlv3.AliasNode:
	default:
	}

	return TraceResult{}, false
}

func findTraceOriginInMapping(
	node *yamlv3.Node,
	parts []string,
	file string,
	anchors map[string]anchorOrigin,
) (TraceResult, bool) {
	for index := 0; index < len(node.Content)-1; index += 2 {
		keyNode := node.Content[index]
		valueNode := node.Content[index+1]

		if keyNode.Value != parts[0] || keyNode.Value == "<<" {
			continue
		}

		if len(parts) == 1 {
			return TraceResult{File: file, Line: keyNode.Line}, true
		}

		return findTraceOrigin(valueNode, parts[1:], file, anchors)
	}

	for index := 0; index < len(node.Content)-1; index += 2 {
		keyNode := node.Content[index]
		valueNode := node.Content[index+1]

		if keyNode.Value != "<<" {
			continue
		}

		if origin, ok := findTraceOriginInMerge(valueNode, parts, anchors); ok {
			return origin, true
		}
	}

	return TraceResult{}, false
}

func findTraceOriginInMerge(
	node *yamlv3.Node,
	parts []string,
	anchors map[string]anchorOrigin,
) (TraceResult, bool) {
	switch node.Kind {
	case yamlv3.AliasNode:
		anchor, exists := anchors[node.Value]
		if !exists {
			return TraceResult{}, false
		}

		return findTraceOrigin(anchor.node, parts, anchor.file, anchors)
	case yamlv3.ScalarNode:
		if !strings.HasPrefix(node.Value, traceAliasPrefix) {
			return TraceResult{}, false
		}

		anchorName := strings.TrimPrefix(node.Value, traceAliasPrefix)

		anchor, exists := anchors[anchorName]
		if !exists {
			return TraceResult{}, false
		}

		return findTraceOrigin(anchor.node, parts, anchor.file, anchors)
	case yamlv3.SequenceNode:
		for _, child := range node.Content {
			if origin, ok := findTraceOriginInMerge(child, parts, anchors); ok {
				return origin, true
			}
		}
	case yamlv3.DocumentNode, yamlv3.MappingNode:
	}

	return TraceResult{}, false
}

func escapeAliasesForTrace(data string) string {
	return traceAliasPattern.ReplaceAllString(data, `${1}"`+traceAliasPrefix+`${2}"`)
}

func displayPath(path string) string {
	if path == "" {
		return path
	}

	relPath, err := filepath.Rel(".", path)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return path
	}

	return relPath
}
