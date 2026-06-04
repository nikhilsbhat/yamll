package yamll

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/nikhilsbhat/yamll/pkg/errors"
)

const (
	TreeOutputText    = "text"
	TreeOutputJSON    = "json"
	TreeOutputDOT     = "dot"
	TreeOutputMermaid = "mermaid"
)

type DependencyTreeNode struct {
	Name     string               `json:"name"`
	Kind     string               `json:"kind,omitempty"`
	Children []DependencyTreeNode `json:"children,omitempty"`
}

func normalizeTreeOutputFormat(format string) string {
	if strings.TrimSpace(format) == "" {
		return TreeOutputText
	}

	return strings.ToLower(strings.TrimSpace(format))
}

func (yamlRoutes YamlRoutes) RenderDependencyTree(rootFile, outputFormat string, noColor, showPatternFiles bool) (string, error) {
	if _, exists := yamlRoutes[rootFile]; !exists {
		return "", &errors.YamllError{Message: fmt.Sprintf("root file '%s' not found in dependency tree", rootFile)}
	}

	tree := yamlRoutes.buildDependencyTreeNode(rootFile, showPatternFiles, make(map[string]bool))

	switch normalizeTreeOutputFormat(outputFormat) {
	case TreeOutputText:
		return renderTreeText(tree, noColor), nil
	case TreeOutputJSON:
		return renderTreeJSON(tree)
	case TreeOutputDOT:
		return renderTreeDOT(tree), nil
	case TreeOutputMermaid:
		return renderTreeMermaid(tree), nil
	default:
		return "", &errors.YamllError{Message: fmt.Sprintf("unsupported tree output format '%s'", outputFormat)}
	}
}

func (yamlRoutes YamlRoutes) buildDependencyTreeNode(name string, showPatternFiles bool, visiting map[string]bool) DependencyTreeNode {
	route := yamlRoutes[name]
	node := DependencyTreeNode{Name: name, Kind: "file"}

	if route == nil {
		return node
	}

	if isPattern(name) {
		node.Kind = "pattern"
	}

	if visiting[name] {
		return node
	}

	visiting[name] = true
	defer delete(visiting, name)

	if showPatternFiles && isPattern(name) {
		patternFiles := make([]string, 0, len(route.SourceFile))

		for _, src := range route.SourceFile {
			if src.Name != "" {
				patternFiles = append(patternFiles, src.Name)
			}
		}

		sort.Strings(patternFiles)

		for _, file := range patternFiles {
			node.Children = append(node.Children, DependencyTreeNode{Name: file, Kind: "matched-file"})
		}
	}

	for _, dep := range route.Dependency {
		if dep == nil {
			continue
		}

		node.Children = append(node.Children, yamlRoutes.buildDependencyTreeNode(dep.Path, showPatternFiles, visiting))
	}

	return node
}

func renderTreeText(root DependencyTreeNode, noColor bool) string {
	color.NoColor = noColor

	var builder strings.Builder

	renderTreeTextNode(&builder, root, "", true)

	return builder.String()
}

func renderTreeTextNode(builder *strings.Builder, node DependencyTreeNode, prefix string, isTail bool) {
	connector := color.GreenString("├── ")
	if isTail {
		connector = color.GreenString("└── ")
	}

	displayName := node.Name

	if node.Kind == "pattern" {
		matchedFiles := 0

		for _, child := range node.Children {
			if child.Kind == "matched-file" {
				matchedFiles++
			}
		}

		if matchedFiles > 0 {
			displayName = fmt.Sprintf("%s (%d files)", node.Name, matchedFiles)
		}
	}

	builder.WriteString(prefix)
	builder.WriteString(connector)
	builder.WriteString(color.MagentaString(displayName))
	builder.WriteByte('\n')

	newPrefix := prefix
	if isTail {
		newPrefix += "    "
	} else {
		newPrefix += color.GreenString("│   ")
	}

	for index, child := range node.Children {
		renderTreeTextNode(builder, child, newPrefix, index == len(node.Children)-1)
	}
}

func renderTreeJSON(root DependencyTreeNode) (string, error) {
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data) + "\n", nil
}

func renderTreeDOT(root DependencyTreeNode) string {
	var (
		builder strings.Builder
		walk    func(node DependencyTreeNode)
		nextID  int
	)

	builder.WriteString("digraph yamll {\n")
	builder.WriteString("  rankdir=TB;\n")

	ids := make(map[string]string)
	emitted := make(map[string]struct{})
	labels := make(map[string]string)

	walk = func(node DependencyTreeNode) {
		nodeID := treeGraphNodeID(node, ids, &nextID)
		writeGraphNode(&builder, emitted, labels, nodeID, node.Name)

		for _, child := range node.Children {
			childID := treeGraphNodeID(child, ids, &nextID)
			writeGraphNode(&builder, emitted, labels, childID, child.Name)
			builder.WriteString("  ")
			builder.WriteString(nodeID)
			builder.WriteString(" -> ")
			builder.WriteString(childID)
			builder.WriteString(";\n")
			walk(child)
		}
	}

	walk(root)
	builder.WriteString("}\n")

	return builder.String()
}

func renderTreeMermaid(root DependencyTreeNode) string {
	var (
		builder strings.Builder
		nextID  int
		walk    func(node DependencyTreeNode)
	)

	builder.WriteString("graph TD\n")

	ids := make(map[string]string)
	emitted := make(map[string]struct{})
	labels := make(map[string]string)

	walk = func(node DependencyTreeNode) {
		nodeID := treeGraphNodeID(node, ids, &nextID)
		writeMermaidNode(&builder, emitted, labels, nodeID, node.Name)

		for _, child := range node.Children {
			childID := treeGraphNodeID(child, ids, &nextID)
			writeMermaidNode(&builder, emitted, labels, childID, child.Name)
			builder.WriteString("  ")
			builder.WriteString(nodeID)
			builder.WriteString(" --> ")
			builder.WriteString(childID)
			builder.WriteByte('\n')
			walk(child)
		}
	}

	walk(root)

	return builder.String()
}

func treeGraphNodeID(node DependencyTreeNode, ids map[string]string, nextID *int) string {
	key := node.Kind + ":" + node.Name
	if id, exists := ids[key]; exists {
		return id
	}

	id := fmt.Sprintf("n%d", *nextID)
	*nextID++
	ids[key] = id

	return id
}

func writeGraphNode(builder *strings.Builder, emitted map[string]struct{}, labels map[string]string, nodeID, name string) {
	if _, exists := emitted[nodeID]; exists {
		return
	}

	emitted[nodeID] = struct{}{}

	builder.WriteString("  ")
	builder.WriteString(nodeID)
	builder.WriteString(" [label=")
	builder.WriteString(strconv.Quote(cachedTreeLabel(labels, name, graphvizLabel)))
	builder.WriteString("];\n")
}

func writeMermaidNode(builder *strings.Builder, emitted map[string]struct{}, labels map[string]string, nodeID, name string) {
	if _, exists := emitted[nodeID]; exists {
		return
	}

	emitted[nodeID] = struct{}{}

	builder.WriteString("  ")
	builder.WriteString(nodeID)
	builder.WriteByte('[')
	builder.WriteString(strconv.Quote(cachedTreeLabel(labels, name, mermaidLabel)))
	builder.WriteString("]\n")
}

func cachedTreeLabel(labels map[string]string, name string, formatter func(string) string) string {
	if label, exists := labels[name]; exists {
		return label
	}

	label := formatter(name)
	labels[name] = label

	return label
}

func graphvizLabel(name string) string {
	return strings.Join(wrapTreeLabel(shortenTreeLabel(name)), "\n")
}

func mermaidLabel(name string) string {
	return strings.Join(wrapTreeLabel(shortenTreeLabel(name)), "<br/>")
}

func shortenTreeLabel(name string) string {
	if strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") {
		return shortenURLLabel(name)
	}

	if filepath.IsAbs(name) {
		if wd, err := os.Getwd(); err == nil {
			if relPath, relErr := filepath.Rel(wd, name); relErr == nil && relPath != ".." && !strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
				return filepath.ToSlash(relPath)
			}
		}

		const pathCount = 4

		return tailPathLabel(filepath.ToSlash(name), pathCount)
	}

	return name
}

func shortenURLLabel(name string) string {
	trimmed := strings.TrimPrefix(strings.TrimPrefix(name, "https://"), "http://")
	trimmed = strings.ReplaceAll(trimmed, "?path=", "/")
	trimmed = strings.ReplaceAll(trimmed, "&", "?")

	return trimmed
}

func wrapTreeLabel(label string) []string {
	const lineWidth = 28

	label = strings.ReplaceAll(label, "\\", "/")
	parts := strings.FieldsFunc(label, func(r rune) bool {
		switch r {
		case '/', '?', '#':
			return true
		default:
			return false
		}
	})

	if len(parts) == 0 {
		return []string{label}
	}

	lines := make([]string, 0, len(parts))
	current := parts[0]

	for _, part := range parts[1:] {
		candidate := current + "/" + part
		if len(candidate) <= lineWidth {
			current = candidate

			continue
		}

		lines = append(lines, current)
		current = part
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

func tailPathLabel(label string, segments int) string {
	parts := strings.Split(strings.Trim(label, "/"), "/")
	if len(parts) <= segments {
		return label
	}

	return strings.Join(parts[len(parts)-segments:], "/")
}
