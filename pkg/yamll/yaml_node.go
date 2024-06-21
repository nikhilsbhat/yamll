package yamll

import (
	"log/slog"

	"github.com/goccy/go-yaml"
	"github.com/nikhilsbhat/yamll/pkg/errors"
	goYaml "gopkg.in/yaml.v3"
)

const yamlIndent = 2

// Explode resolves and substitutes all anchors and aliases in the given YAML.
func (yml Yaml) Explode(log *slog.Logger) (Yaml, error) {
	var v interface{}
	if err := yaml.UnmarshalWithOptions([]byte(yml), &v, yaml.UseOrderedMap(), yaml.DisallowDuplicateKey()); err != nil {
		return "", err
	}

	yamlOUT, err := yaml.MarshalWithOptions(v, yaml.Indent(yamlIndent), yaml.IndentSequence(true), yaml.UseLiteralStyleIfMultiline(true))
	if err != nil {
		return "", err
	}

	var data map[string]interface{}
	if err = goYaml.Unmarshal(yamlOUT, &data); err != nil {
		log.Error("syntax validation of exploded yaml errored", slog.Any("error", err))

		return "", &errors.YamlError{Message: "exploded yaml is invalid"}
	}

	return Yaml(yamlOUT), nil
}
