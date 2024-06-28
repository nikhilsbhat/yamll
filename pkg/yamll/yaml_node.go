package yamll

import (
	"log/slog"
	"strings"

	"github.com/goccy/go-yaml"
)

const yamlIndent = 2

// Explode resolves and substitutes all anchors and aliases in the given YAML.
func (yml Yaml) Explode(log *slog.Logger) (Yaml, error) {
	anchorRefs := strings.NewReader(string(yml))

	var value map[string]interface{}

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

	if err := yaml.UnmarshalWithOptions([]byte(yml), &value, decodeOpts...); err != nil {
		log.Error("errored while serializing yaml")

		return "", err
	}

	yamlOUT, err := yaml.MarshalWithOptions(value, encodeOpts...)
	if err != nil {
		log.Error("errored while de-serializing yaml")

		return "", err
	}

	return Yaml(yamlOUT), nil
}
