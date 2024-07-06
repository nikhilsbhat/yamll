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
	yamlDataS := strings.Split(string(yml), "---")

	var yamlFilesBuilder strings.Builder

	for _, yamlData := range yamlDataS {
		yamlData = strings.TrimSpace(yamlData)
		if len(yamlData) == 0 {
			continue
		}

		var sourceMetadata string
		if scanner := bufio.NewScanner(strings.NewReader(yamlData)); scanner.Scan() {
			sourceMetadata = scanner.Text()
		}

		yamlMap := make(Data)

		anchorRefs := strings.NewReader(string(yml))

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
