package yamll

import (
	"github.com/goccy/go-yaml"
)

const yamlIndent = 2

func (yml Yaml) Explode() (Yaml, error) {
	jsonOUT, err := yaml.YAMLToJSON([]byte(yml))
	if err != nil {
		return "", err
	}

	var v interface{}
	if err := yaml.UnmarshalWithOptions(jsonOUT, &v, yaml.UseOrderedMap()); err != nil {
		return "", err
	}

	yamlOUT, err := yaml.MarshalWithOptions(v, yaml.Indent(yamlIndent), yaml.IndentSequence(true))
	if err != nil {
		return "", err
	}

	return Yaml(yamlOUT), nil
}
