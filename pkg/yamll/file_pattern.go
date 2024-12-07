package yamll

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/nikhilsbhat/yamll/pkg/errors"
)

type File struct {
	Name string
	Data string
}

// FilePattern reads the data from the Files matching the pattern import.
func (dependency *Dependency) FilePattern(log *slog.Logger) (File, error) {
	log.Debug("Since the path is a pattern, the filenames matching the pattern will be hidden from the tree, import, and build commands. " +
		"Instead, the data from all files matching the pattern will be consolidated under the file pattern.")

	absPatternPath, err := filepath.Abs(filepath.Dir(dependency.Path))
	if err != nil {
		return File{}, err
	}

	absFilePattern := filepath.Join(absPatternPath, filepath.Base(dependency.Path))

	filesMatching, err := filepath.Glob(absFilePattern)
	if err != nil {
		return File{}, &errors.YamllError{Message: fmt.Sprintf("error matching pattern: '%v'", err)}
	}

	var yamlFilesData string

	log.Debug("the files matching the pattern are", slog.Any("pattern", dependency.Path), slog.Any("files-matched", filesMatching))

	for _, fileMatching := range filesMatching {
		yamlFileData, err := os.ReadFile(fileMatching)
		if err != nil {
			return File{}, &errors.YamllError{Message: fmt.Sprintf("reading YAML dependency errored with: '%v'", err)}
		}

		yamlFilesData = yamlFilesData + "\n" + string(yamlFileData)
	}

	return File{Name: absFilePattern, Data: yamlFilesData}, nil
}
