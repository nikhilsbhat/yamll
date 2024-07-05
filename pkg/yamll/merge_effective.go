package yamll

import (
	"fmt"
	"log"
	"strings"

	"dario.cat/mergo"
	"github.com/goccy/go-yaml"
	"github.com/nikhilsbhat/yamll/pkg/errors"
)

// Data is a data type that holds de-serialised yaml data.
type Data map[string]interface{}

// EffectiveMerge merges multiple yaml contents effectively.
func (yml Yaml) EffectiveMerge() (Yaml, error) {
	yamlMapMerged := make(Data)

	yamlDataS := strings.Split(string(yml), "---")

	for _, yamlData := range yamlDataS {
		yamlData = strings.TrimSpace(yamlData)
		if len(yamlData) == 0 {
			continue
		}

		yamlMap := make(Data)

		anchorRefs := strings.NewReader(string(yml))

		if err := yaml.UnmarshalWithOptions([]byte(yamlData), &yamlMap, yaml.ReferenceReaders(anchorRefs)); err != nil {
			log.Fatal(err)
		}

		if err := mergeStructs(&yamlMapMerged, yamlMap); err != nil {
			return "", &errors.YamllError{Message: fmt.Sprintf("error merging YAML files: %v", err)}
		}
	}

	yamlOut, err := yaml.MarshalWithOptions(yamlMapMerged, yaml.Indent(yamlIndent), yaml.IndentSequence(true))
	if err != nil {
		return "", err
	}

	return Yaml(yamlOut), nil
}

func mergeStructs(dest, src interface{}) error {
	if err := mergo.Merge(dest, src, mergo.WithOverride); err != nil {
		return err
	}

	destMap, destIsMap := dest.(map[string]interface{})
	srcMap, srcIsMap := src.(map[string]interface{})

	if destIsMap && srcIsMap {
		for key, srcVal := range srcMap {
			if destVal, ok := destMap[key]; ok {
				if err := mergeStructs(destVal, srcVal); err != nil {
					return err
				}
			} else {
				destMap[key] = srcVal
			}
		}
	}

	return nil
}
