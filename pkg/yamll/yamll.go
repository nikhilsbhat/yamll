package yamll

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"dario.cat/mergo"
	"gopkg.in/yaml.v3"
)

// YamlData holds information of yaml file and its dependency tree.
type YamlData struct {
	Root       bool                   `json:"root,omitempty" yaml:"root,omitempty"`
	Imported   bool                   `json:"imported,omitempty" yaml:"imported,omitempty"`
	File       string                 `json:"file,omitempty" yaml:"file,omitempty"`
	DataRaw    string                 `json:"data_raw,omitempty" yaml:"data_raw,omitempty"`
	Dependency []string               `json:"dependency,omitempty" yaml:"dependency,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty" yaml:"data,omitempty"`
}

// Config holds the information of yaml files to be parsed.
type Config struct {
	Files []string
	Root  bool
}

// Yaml identifies the YAML imports and merges them to create a single comprehensive YAML file.
// These imports function similarly to importing libraries in a programming language.
func (cfg *Config) Yaml() (string, error) {
	dependencyRoutes, err := cfg.ResolveDependencies(make(map[string]*YamlData), cfg.Files...)
	if err != nil {
		return "", fmt.Errorf("fetching delendency tree errored with: '%s'", err)
	}

	var importData map[string]interface{}

	finalData, err := MergeData(importData, dependencyRoutes)
	if err != nil {
		return "", err
	}

	fileData, err := yaml.Marshal(finalData)
	if err != nil {
		return "", fmt.Errorf("marshalling data to YAML errored with: '%v'", err)
	}

	return string(fileData), nil
}

// ResolveDependencies addresses the dependencies of YAML imports specified in the YAML files.
func (cfg *Config) ResolveDependencies(routes map[string]*YamlData, yamlFilesPath ...string) (map[string]*YamlData, error) {
	var rootFile bool
	if !cfg.Root {
		rootFile = true
	}
	for fileHierarchy, yamlFilePath := range yamlFilesPath {
		absYamlFilePath, err := filepath.Abs(yamlFilePath)
		if err != nil {
			return nil, err
		}

		yamlFileData, err := os.ReadFile(absYamlFilePath)
		if err != nil {
			return nil, fmt.Errorf("reading YAML dependency errored with: '%v'", err)
		}

		dependencies := make([]string, 0)
		stringReader := strings.NewReader(string(yamlFileData))

		scanner := bufio.NewScanner(stringReader)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "##++") {
				dependencies = append(dependencies, getDependencyData(line))
			}
		}

		if fileHierarchy == 0 && !cfg.Root {
			cfg.Root = true
		}

		var marshalledData map[string]interface{}
		if err = yaml.Unmarshal(yamlFileData, &marshalledData); err != nil {
			return nil, err
		}

		routes[yamlFilePath] = &YamlData{Root: rootFile, File: yamlFilePath, DataRaw: string(yamlFileData), Data: marshalledData, Dependency: dependencies}

		if len(dependencies) != 0 {
			dependencyRoutes, err := cfg.ResolveDependencies(routes, dependencies...)
			if err != nil {
				return nil, err
			}
			if err = mergo.Merge(&routes, dependencyRoutes, mergo.WithOverride); err != nil {
				return nil, fmt.Errorf("error merging YAML files: %v", err)
			}
		}
	}

	return routes, nil
}

// MergeData combines the YAML file data according to the hierarchy.
func MergeData(src map[string]interface{}, data map[string]*YamlData) (map[string]interface{}, error) {
	for file, fileData := range data {
		if !fileData.Root {
			continue
		}

		out, err := Merge(src, data, file)
		if err != nil {
			return nil, err
		}

		if err = mergo.Merge(&out, fileData.Data, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("error merging YAML files: %v", err)
		}
		log.Printf("root file '%s' was imported successfully", file)

		src = out
	}

	return src, nil
}

func Merge(src map[string]interface{}, data map[string]*YamlData, file string) (map[string]interface{}, error) {
	for _, dependency := range data[file].Dependency {
		if data[dependency].Imported {
			log.Printf("file '%s' already imported hence skipping", dependency)
			continue
		}

		out, err := Merge(src, data, dependency)
		if err != nil {
			return nil, err
		}

		if err := mergo.Merge(&src, out, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("error merging YAML files: %v", err)
		}
	}

	if !data[file].Imported && !data[file].Root {
		if err := mergo.Merge(&src, data[file].Data, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("error merging YAML files: %v", err)
		}

		data[file].Imported = true
		log.Printf("file '%s' was imported successfully", file)
	}

	return src, nil
}

func getDependencyData(dependency string) string {
	imports := strings.Split(dependency, ";")
	runeSlice := []rune(imports[0])

	return string(runeSlice[4:])
}

func New(path ...string) *Config {
	return &Config{Files: path}
}
