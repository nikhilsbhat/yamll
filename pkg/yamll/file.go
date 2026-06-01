package yamll

import (
	"crypto/sha256"
	"encoding/hex"
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

	sum := sha256.Sum256(yamlFileData)

	return File{Name: absYamlFilePath, Data: string(yamlFileData), Meta: FileMeta{SHA256: hex.EncodeToString(sum[:])}}, nil
}
