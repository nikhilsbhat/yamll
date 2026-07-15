package yamll

import (
	stdErrors "errors"
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
	Path   string `json:"path,omitempty" yaml:"path,omitempty"`
	Origin string `json:"origin,omitempty" yaml:"origin,omitempty"`
	File   string `json:"file,omitempty" yaml:"file,omitempty"`
	Line   int    `json:"line,omitempty" yaml:"line,omitempty"`
}

type ExplainResult struct {
	Path        string        `json:"path" yaml:"path"`
	FinalValue  string        `json:"final_value" yaml:"final_value"`
	FinalOrigin TraceResult   `json:"final_origin" yaml:"final_origin"`
	Steps       []ExplainStep `json:"steps" yaml:"steps"`
	MergeRule   string        `json:"merge_rule" yaml:"merge_rule"`
}

type ExplainStep struct {
	Origin string `json:"origin" yaml:"origin"`
	File   string `json:"file" yaml:"file"`
	Line   int    `json:"line" yaml:"line"`
	Value  string `json:"value" yaml:"value"`
	Source string `json:"source" yaml:"source"`
	Action string `json:"action" yaml:"action"`
}

type anchorOrigin struct {
	node *yamlv3.Node
	file string
}

type explainCandidate struct {
	file   string
	line   int
	node   *yamlv3.Node
	source string
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
			if stdErrors.Is(err, &errors.YamlEmptyError{}) {
				continue
			}

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

func (cfg *Config) Explain(path string) (ExplainResult, error) {
	if len(cfg.Files) == 0 {
		return ExplainResult{}, &errors.YamllError{Message: "explain requires a root file"}
	}

	cfg.Root = false

	routes, err := cfg.ResolveDependencies(make(map[string]*YamlData), cfg.Files...)
	if err != nil {
		return ExplainResult{}, &errors.YamllError{Message: fmt.Sprintf("fetching dependency tree errored with: '%v'", err)}
	}

	rootFile := cfg.Files[0].Path

	rootRoute, exists := routes[rootFile]
	if !exists {
		return ExplainResult{}, &errors.YamllError{Message: fmt.Sprintf("root file '%s' not found in dependency tree", rootFile)}
	}

	parts := strings.Split(strings.TrimSpace(path), ".")
	if len(parts) == 0 || parts[0] == "" {
		return ExplainResult{}, &errors.YamllError{Message: "explain path cannot be empty"}
	}

	yamlRoutes := YamlRoutes(routes)

	anchors, err := yamlRoutes.collectAnchors()
	if err != nil {
		return ExplainResult{}, err
	}

	var (
		candidates  = make([]explainCandidate, 0)
		finalOrigin TraceResult
	)

	for _, sourceFile := range rootRoute.SourceFile {
		node, err := parseYAMLSource(sourceFile.Data)
		if err != nil {
			if stdErrors.Is(err, &errors.YamlEmptyError{}) {
				continue
			}

			return ExplainResult{}, err
		}

		candidates = append(candidates, findExplainCandidates(node, parts, sourceFile.Name, anchors)...)

		if finalOrigin.Origin == "" {
			origin, ok := findTraceOrigin(node, parts, sourceFile.Name, anchors)
			if ok {
				origin.File = displayPath(origin.File)
				origin.Path = path
				origin.Origin = fmt.Sprintf("%s:%d", origin.File, origin.Line)
				finalOrigin = origin
			}
		}
	}

	if finalOrigin.Origin == "" {
		return ExplainResult{}, &errors.YamllError{Message: fmt.Sprintf("path '%s' not found in generated YAML", path)}
	}

	steps, finalValue, renderFinalValue := buildExplainSteps(candidates, finalOrigin, anchors)

	if renderFinalValue {
		renderedValue, ok := findRenderedExplainValue(yamlRoutes, parts)
		if ok {
			finalValue = renderedValue
		}
	}

	return ExplainResult{
		Path:        path,
		FinalValue:  finalValue,
		FinalOrigin: finalOrigin,
		Steps:       steps,
		MergeRule:   "direct keys override merged values; root content is evaluated after imported anchors",
	}, nil
}

func (yamlRoutes YamlRoutes) collectAnchors() (map[string]anchorOrigin, error) {
	anchors := make(map[string]anchorOrigin)

	for _, file := range yamlRoutes.OrderedFiles() {
		route := yamlRoutes[file]
		for _, sourceFile := range route.SourceFile {
			node, err := parseYAMLSource(sourceFile.Data)
			if err != nil {
				if stdErrors.Is(err, &errors.YamlEmptyError{}) {
					continue
				}

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
		return nil, &errors.YamlEmptyError{Message: "empty YAML document"}
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

func findExplainCandidates(
	node *yamlv3.Node,
	parts []string,
	file string,
	anchors map[string]anchorOrigin,
) []explainCandidate {
	if node == nil {
		return nil
	}

	if len(parts) == 0 {
		return []explainCandidate{{file: file, line: node.Line, node: node, source: "direct"}}
	}

	if node.Kind == yamlv3.AliasNode {
		anchor, exists := anchors[node.Value]
		if !exists {
			return nil
		}

		return findExplainCandidates(anchor.node, parts, anchor.file, anchors)
	}

	if node.Kind == yamlv3.ScalarNode && strings.HasPrefix(node.Value, traceAliasPrefix) {
		anchorName := strings.TrimPrefix(node.Value, traceAliasPrefix)

		anchor, exists := anchors[anchorName]
		if !exists {
			return nil
		}

		return findExplainCandidates(anchor.node, parts, anchor.file, anchors)
	}

	switch node.Kind {
	case yamlv3.MappingNode:
		return findExplainCandidatesInMapping(node, parts, file, anchors)
	case yamlv3.SequenceNode:
		index, err := strconv.Atoi(parts[0])
		if err != nil || index < 0 || index >= len(node.Content) {
			return nil
		}

		return findExplainCandidates(node.Content[index], parts[1:], file, anchors)
	case yamlv3.DocumentNode, yamlv3.ScalarNode, yamlv3.AliasNode:
	default:
	}

	return nil
}

func findExplainCandidatesInMapping(
	node *yamlv3.Node,
	parts []string,
	file string,
	anchors map[string]anchorOrigin,
) []explainCandidate {
	candidates := make([]explainCandidate, 0)

	for index := 0; index < len(node.Content)-1; index += 2 {
		keyNode := node.Content[index]
		valueNode := node.Content[index+1]

		if keyNode.Value != "<<" {
			continue
		}

		candidates = append(candidates, findExplainCandidatesInMerge(valueNode, parts, anchors)...)
	}

	for index := 0; index < len(node.Content)-1; index += 2 {
		keyNode := node.Content[index]
		valueNode := node.Content[index+1]

		if keyNode.Value != parts[0] || keyNode.Value == "<<" {
			continue
		}

		if len(parts) == 1 {
			candidates = append(candidates, explainCandidate{
				file:   file,
				line:   keyNode.Line,
				node:   resolveExplainNode(valueNode, anchors),
				source: "direct",
			})

			continue
		}

		candidates = append(candidates, findExplainCandidates(valueNode, parts[1:], file, anchors)...)
	}

	return candidates
}

func findExplainCandidatesInMerge(
	node *yamlv3.Node,
	parts []string,
	anchors map[string]anchorOrigin,
) []explainCandidate {
	switch node.Kind {
	case yamlv3.AliasNode:
		anchor, exists := anchors[node.Value]
		if !exists {
			return nil
		}

		return markExplainCandidatesSource(findExplainCandidates(anchor.node, parts, anchor.file, anchors), "merge")
	case yamlv3.ScalarNode:
		if !strings.HasPrefix(node.Value, traceAliasPrefix) {
			return nil
		}

		anchorName := strings.TrimPrefix(node.Value, traceAliasPrefix)

		anchor, exists := anchors[anchorName]
		if !exists {
			return nil
		}

		return markExplainCandidatesSource(findExplainCandidates(anchor.node, parts, anchor.file, anchors), "merge")
	case yamlv3.SequenceNode:
		candidates := make([]explainCandidate, 0)
		for _, child := range node.Content {
			candidates = append(candidates, findExplainCandidatesInMerge(child, parts, anchors)...)
		}

		return candidates
	case yamlv3.DocumentNode, yamlv3.MappingNode:
	}

	return nil
}

func markExplainCandidatesSource(candidates []explainCandidate, source string) []explainCandidate {
	for index := range candidates {
		candidates[index].source = source
	}

	return candidates
}

func buildExplainSteps(
	candidates []explainCandidate,
	finalOrigin TraceResult,
	anchors map[string]anchorOrigin,
) ([]ExplainStep, string, bool) {
	steps := make([]ExplainStep, 0, len(candidates))
	finalValue := ""
	renderFinalValue := false

	for _, candidate := range candidates {
		file := displayPath(candidate.file)
		origin := fmt.Sprintf("%s:%d", file, candidate.line)
		node := resolveExplainNode(candidate.node, anchors)
		value := formatExplainValue(node)
		action := "overridden"

		if origin == finalOrigin.Origin {
			action = "selected"
			finalValue = value
			renderFinalValue = node != nil && node.Kind != yamlv3.ScalarNode
		}

		steps = append(steps, ExplainStep{
			Origin: origin,
			File:   file,
			Line:   candidate.line,
			Value:  value,
			Source: candidate.source,
			Action: action,
		})
	}

	return steps, finalValue, renderFinalValue
}

func findRenderedExplainValue(routes YamlRoutes, parts []string) (string, bool) {
	rendered, err := routes.Build()
	if err != nil {
		return "", false
	}

	var node yamlv3.Node
	if err = yamlv3.Unmarshal([]byte(rendered), &node); err != nil || len(node.Content) == 0 {
		return "", false
	}

	valueNode, ok := findRenderedExplainNode(node.Content[0], parts)
	if !ok {
		return "", false
	}

	return formatExplainValue(valueNode), true
}

func findRenderedExplainNode(node *yamlv3.Node, parts []string) (*yamlv3.Node, bool) {
	if node == nil {
		return nil, false
	}

	if len(parts) == 0 {
		return node, true
	}

	switch node.Kind {
	case yamlv3.MappingNode:
		for index := 0; index < len(node.Content)-1; index += 2 {
			keyNode := node.Content[index]
			valueNode := node.Content[index+1]

			if keyNode.Value == parts[0] {
				return findRenderedExplainNode(valueNode, parts[1:])
			}
		}
	case yamlv3.SequenceNode:
		index, err := strconv.Atoi(parts[0])
		if err != nil || index < 0 || index >= len(node.Content) {
			return nil, false
		}

		return findRenderedExplainNode(node.Content[index], parts[1:])
	case yamlv3.DocumentNode:
		if len(node.Content) == 0 {
			return nil, false
		}

		return findRenderedExplainNode(node.Content[0], parts)
	case yamlv3.ScalarNode, yamlv3.AliasNode:
	}

	return nil, false
}

func resolveExplainNode(node *yamlv3.Node, anchors map[string]anchorOrigin) *yamlv3.Node {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yamlv3.AliasNode:
		if anchor, exists := anchors[node.Value]; exists {
			return anchor.node
		}
	case yamlv3.ScalarNode:
		if anchorName, found := strings.CutPrefix(node.Value, traceAliasPrefix); found {
			if anchor, exists := anchors[anchorName]; exists {
				return anchor.node
			}
		}
	case yamlv3.DocumentNode, yamlv3.SequenceNode, yamlv3.MappingNode:
	}

	return node
}

func formatExplainValue(node *yamlv3.Node) string {
	if node == nil {
		return ""
	}

	if node.Kind == yamlv3.ScalarNode {
		if anchorName, found := strings.CutPrefix(node.Value, traceAliasPrefix); found {
			return "*" + anchorName
		}

		return node.Value
	}

	out, err := yamlv3.Marshal(node)
	if err != nil {
		return ""
	}

	return cleanExplainValue(strings.TrimSpace(string(out)))
}

func cleanExplainValue(value string) string {
	var builder strings.Builder

	builder.Grow(len(value))

	for line := range strings.SplitSeq(value, "\n") {
		line = strings.Replace(line, "!!merge <<:", "<<:", 1)
		if strings.Contains(line, traceAliasPrefix) {
			line = strings.ReplaceAll(line, `"`+traceAliasPrefix, "*")
			line = strings.ReplaceAll(line, traceAliasPrefix, "*")
			line = strings.TrimSuffix(line, `"`)
		}

		builder.WriteString(line)
		builder.WriteByte('\n')
	}

	return strings.TrimSuffix(builder.String(), "\n")
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
