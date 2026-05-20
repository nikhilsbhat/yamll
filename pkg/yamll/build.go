package yamll

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/nikhilsbhat/yamll/pkg/errors"
)

func (yamlRoutes YamlRoutes) Build() (Yaml, error) {
	anchorRefData := dedupeAnchorReferences(yamlRoutes.getRawData())

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
