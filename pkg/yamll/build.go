package yamll

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/nikhilsbhat/yamll/pkg/errors"
	yamlv3 "gopkg.in/yaml.v3"
)

var (
	mergeScalarAliasPattern = regexp.MustCompile(`(?m)^\s*<<:\s*\*([A-Za-z0-9_-]+)\s*$`)
	mergeFlowAliasPattern   = regexp.MustCompile(`(?m)^\s*<<:\s*\[([^\]]+)\]\s*$`)
	flowAliasPattern        = regexp.MustCompile(`\*([A-Za-z0-9_-]+)`)
)

func (yamlRoutes YamlRoutes) Build() (Yaml, error) {
	anchorRefData := dedupeAnchorReferences(yamlRoutes.getRawData())

	anchorKinds, err := collectAnchorKinds(anchorRefData)
	if err != nil {
		return "", err
	}

	var output []byte

	for _, file := range yamlRoutes.OrderedFiles() {
		dependencyRoute := yamlRoutes[file]
		if !dependencyRoute.Root {
			continue
		}

		var yamlMap yaml.MapSlice

		decodeOpts := []yaml.DecodeOption{
			yaml.UseOrderedMap(),
			yaml.Strict(),
			yaml.ReferenceReaders(strings.NewReader(anchorRefData)),
		}

		encodeOpts := []yaml.EncodeOption{
			yaml.Indent(yamlIndent),
			yaml.IndentSequence(true),
			yaml.UseLiteralStyleIfMultiline(true),
		}

		if err := validateMergeAliases(dependencyRoute.DataRaw, anchorKinds); err != nil {
			return "", err
		}

		if err := yaml.UnmarshalWithOptions([]byte(dependencyRoute.DataRaw), &yamlMap, decodeOpts...); err != nil {
			return "", &errors.YamllError{Message: fmt.Sprintf("error deserialising YAML file: %v", err)}
		}

		yamlOut, err := yaml.MarshalWithOptions(yamlMap, encodeOpts...)
		if err != nil {
			return "", &errors.YamllError{Message: fmt.Sprintf("serialising YAML file %s errored : %v", dependencyRoute.File, err)}
		}

		output = yamlOut
	}

	return Yaml(output), nil
}

func collectAnchorKinds(yamlData string) (map[string]yamlv3.Kind, error) {
	var (
		doc  yamlv3.Node
		walk func(name *yamlv3.Node)
	)

	if err := yamlv3.Unmarshal([]byte(yamlData), &doc); err != nil {
		return nil, &errors.YamllError{Message: fmt.Sprintf("parsing YAML anchors errored with: '%v'", err)}
	}

	kinds := make(map[string]yamlv3.Kind)

	walk = func(name *yamlv3.Node) {
		if name == nil {
			return
		}

		if name.Anchor != "" {
			if _, exists := kinds[name.Anchor]; !exists {
				kinds[name.Anchor] = name.Kind
			}
		}

		for _, c := range name.Content {
			walk(c)
		}
	}

	walk(&doc)

	return kinds, nil
}

func validateMergeAliases(yamlData string, anchorKinds map[string]yamlv3.Kind) error {
	aliases := make([]string, 0)

	for _, m := range mergeScalarAliasPattern.FindAllStringSubmatch(yamlData, -1) {
		aliases = append(aliases, m[1])
	}

	for _, m := range mergeFlowAliasPattern.FindAllStringSubmatch(yamlData, -1) {
		for _, a := range flowAliasPattern.FindAllStringSubmatch(m[1], -1) {
			aliases = append(aliases, a[1])
		}
	}

	for _, alias := range aliases {
		kind, exists := anchorKinds[alias]
		if !exists {
			// Could be defined in the same file without being in the merged anchor data.
			// Let the YAML parser surface the missing anchor error.
			continue
		}

		if kind != yamlv3.MappingNode {
			return &errors.YamllError{
				Message: fmt.Sprintf("invalid YAML merge: '<<: *%s' refers to a non-mapping anchor. Use 'key: *%s' for sequences/scalars, or define &%s as a mapping",
					alias, alias, alias),
			}
		}
	}

	return nil
}

func (yamlRoutes YamlRoutes) getRawData() string {
	var builder strings.Builder

	for _, file := range yamlRoutes.OrderedFiles() {
		dependencyRoute := yamlRoutes[file]

		builder.WriteString("---\n")
		builder.WriteString(dependencyRoute.DataRaw)
		builder.WriteString("\n")
	}

	return builder.String()
}
