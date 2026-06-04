package yamll

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/nikhilsbhat/yamll/pkg/errors"
)

const yamlIndent = 2

// Explode resolves and substitutes all anchors and aliases in the given YAML.
func (yml Yaml) Explode() (Yaml, error) {
	rawYAML := string(yml)

	var yamlFilesBuilder strings.Builder

	yamlFilesBuilder.Grow(len(rawYAML) + len(rawYAML)/4)

	for yamlData := range strings.SplitSeq(rawYAML, "---") {
		yamlData = strings.TrimSpace(yamlData)
		if len(yamlData) == 0 {
			continue
		}

		var sourceMetadata string
		if scanner := bufio.NewScanner(strings.NewReader(yamlData)); scanner.Scan() {
			sourceMetadata = scanner.Text()
		}

		var yamlMap yaml.MapSlice

		anchorRefs := strings.NewReader(rawYAML)

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

		if err := yaml.UnmarshalWithOptions([]byte(yamlData), &yamlMap, decodeOpts...); err != nil {
			return "", &errors.YamllError{Message: fmt.Sprintf("deserialising YAML file %s errored : %v", sourceMetadata, err)}
		}

		yamlOut, err := yaml.MarshalWithOptions(yamlMap, encodeOpts...)
		if err != nil {
			return "", &errors.YamllError{Message: fmt.Sprintf("serialising YAML file %s errored : %v", sourceMetadata, err)}
		}

		yamlFilesBuilder.WriteString("---\n")
		yamlFilesBuilder.WriteString(sourceMetadata)
		yamlFilesBuilder.WriteString("\n")
		yamlFilesBuilder.Write(yamlOut)
		yamlFilesBuilder.WriteString("\n")
	}

	return Yaml(yamlFilesBuilder.String()), nil
}
