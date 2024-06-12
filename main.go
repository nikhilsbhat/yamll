package main

import (
	"fmt"
	"github.com/nikhilsbhat/yamll/pkg/yamll"
	"log"
	"os"

	"dario.cat/mergo"
	"gopkg.in/yaml.v3"
)

func main() {
	fmt.Println(yamll.Yaml("internal/fixtures/import.yaml"))
}
func main2() {
	baseFile := "internal/fixtures/base.yaml"
	importFile := "internal/fixtures/import.yaml"
	outputFile := "merged.yaml"

	// Read base YAML file
	baseData, err := readYAML(baseFile)
	if err != nil {
		log.Fatalf("Error reading base YAML file: %v", err)
	}

	// Read import YAML file
	importData, err := readYAML(importFile)
	if err != nil {
		log.Fatalf("Error reading import YAML file: %v", err)
	}

	// Merge the YAML contents
	if err := mergo.Merge(&baseData, importData, mergo.WithOverride); err != nil {
		log.Fatalf("Error merging YAML files: %v", err)
	}

	// Write the merged YAML to a new file
	if err := writeYAML(outputFile, baseData); err != nil {
		log.Fatalf("Error writing merged YAML file: %v", err)
	}

	fmt.Printf("Merged YAML written to %s\n", outputFile)
}

// readYAML reads a YAML file into a map
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

// writeYAML writes a map to a YAML file
func writeYAML(fileName string, data map[string]interface{}) error {
	fileData, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(fileName, fileData, os.ModePerm)
}
