package yamll

import (
	"fmt"
	"regexp"
	"strings"

	"dario.cat/mergo"
	"github.com/goccy/go-yaml"
	"github.com/nikhilsbhat/yamll/pkg/errors"
)

// Data is a data type that holds de-serialised YAML data.
type Data map[string]any

var anchorPattern = regexp.MustCompile(`(^|[\s\[{,])&([A-Za-z0-9_-]+)`)

// EffectiveMerge merges multiple YAML contents effectively.
func (yml Yaml) EffectiveMerge() (Yaml, error) {
	yamlMapMerged := make(Data)
	anchorRefData := dedupeAnchorReferences(string(yml))

	for yamlData := range strings.SplitSeq(string(yml), "---") {
		yamlData = strings.TrimSpace(yamlData)
		if len(yamlData) == 0 {
			continue
		}

		yamlMap := make(Data)

		if err := yaml.UnmarshalWithOptions([]byte(yamlData), &yamlMap, yaml.ReferenceReaders(strings.NewReader(anchorRefData))); err != nil {
			return "", &errors.YamllError{Message: fmt.Sprintf("error deserialising YAML file: %v", err)}
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

func dedupeAnchorReferences(yamlData string) string {
	seen := make(map[string]int)

	return anchorPattern.ReplaceAllStringFunc(yamlData, func(anchorMatch string) string {
		prefix, name, found := strings.Cut(anchorMatch, "&")
		if !found {
			return anchorMatch
		}

		seen[name]++

		if seen[name] == 1 {
			return anchorMatch
		}

		return fmt.Sprintf("%s&%s_yamll_duplicate_%d", prefix, name, seen[name])
	})
}

func mergeStructs(dest, src any) error {
	if err := mergo.Merge(dest, src, mergo.WithOverride); err != nil {
		return err
	}

	destMap, destIsMap := dest.(map[string]any)
	srcMap, srcIsMap := src.(map[string]any)

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
