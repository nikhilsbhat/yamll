package yamll

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/nikhilsbhat/yamll/pkg/errors"
)

func (yamlRoutes YamlRoutes) Build() (Yaml, error) {
	anchorRefs := strings.NewReader(yamlRoutes.getRawData())

	var output []byte

	for _, dependencyRoute := range yamlRoutes {
		if !dependencyRoute.Root {
			continue
		}

		yamlMap := make(Data)

		decodeOpts := []yaml.DecodeOption{
			yaml.UseOrderedMap(),
			yaml.Strict(),
			yaml.ReferenceReaders(anchorRefs),
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
	var rawData string

	for _, dependencyRoute := range yamlRoutes {
		rawData += fmt.Sprintf("---\n%s\n", dependencyRoute.DataRaw)
	}

	return rawData
}
