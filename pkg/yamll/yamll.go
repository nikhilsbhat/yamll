package yamll

import (
	"bufio"
	"dario.cat/mergo"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

type YamlRoute struct {
	Parent []string
	Child  string
}

type YamlData struct {
	File       string
	Data       string
	Dependency []string
}

func Yaml(path ...string) (string, error) {
	routes := make([]*YamlRoute, 0)
	for _, childPath := range path {
		routes = append(routes, &YamlRoute{Child: childPath})
	}

	for _, route := range routes {
		fileData, err := os.ReadFile(route.Child)
		if err != nil {
			return "", fmt.Errorf("reading child data failed with: '%v'", err)
		}

		route.Parent = dependency(string(fileData))
	}

	var importData map[string]interface{}
	for _, route := range routes {
		for _, parent := range route.Parent {
			currentData, err := readYAML(parent)
			if err != nil {
				return "", fmt.Errorf("error reading import YAML file: '%v'", err)
			}
			if err = mergo.Merge(&importData, currentData, mergo.WithOverride); err != nil {
				return "", fmt.Errorf("error merging YAML files: '%v'", err)
			}
		}

		childData, err := readYAML(route.Child)
		if err != nil {
			return "", fmt.Errorf("error reading import YAML file: '%v'", err)
		}
		if err = mergo.Merge(&importData, childData, mergo.WithOverride); err != nil {
			return "", fmt.Errorf("error merging YAML files: %v", err)
		}
	}

	fileData, err := yaml.Marshal(importData)
	if err != nil {
		return "", fmt.Errorf("marshalling data to YAML errored with: '%v'", err)
	}

	return string(fileData), nil
}

func dependency(data string) []string {
	dependencies := make([]string, 0)
	stringReader := strings.NewReader(data)

	scanner := bufio.NewScanner(stringReader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "##++") {
			imports := strings.Split(line, ";")
			runeSlice := []rune(imports[0])
			dependencies = append(dependencies, string(runeSlice[4:]))
		}
	}

	return dependencies
}

func resolveDependencies() {

}

func readYAML(fileName string) (map[string]interface{}, error) {
	fileData, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal(fileData, &data); err != nil {
		return nil, err
	}

	return data, nil
}
