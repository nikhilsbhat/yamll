package yamll

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/nikhilsbhat/yamll/pkg/errors"
)

// File reads the data from the File import.
func (dependency *Dependency) File(_ *slog.Logger) (File, error) {
	absYamlFilePath, err := filepath.Abs(dependency.Path)
	if err != nil {
		return File{}, err
	}

	yamlFileData, err := os.ReadFile(absYamlFilePath)
	if err != nil {
		return File{}, &errors.YamllError{Message: fmt.Sprintf("reading YAML dependency errored with: '%v'", err)}
	}

	return File{Name: absYamlFilePath, Data: string(yamlFileData)}, nil
}
