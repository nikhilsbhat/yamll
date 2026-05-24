package yamll

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/nikhilsbhat/yamll/pkg/errors"
)

type File struct {
	Name   string
	Data   string
	Source []File
}

// FilePattern reads the data from the Files matching the pattern import.
func (dependency *Dependency) FilePattern(log *slog.Logger) (File, error) {
	log.Debug("Since the path is a pattern, the filenames matching the pattern will be hidden from the tree, import, and build commands. " +
		"Instead, the data from all files matching the pattern will be consolidated under the file pattern.")

	absFilePattern, filesMatching, err := dependency.filesMatchingPattern()
	if err != nil {
		return File{}, err
	}

	var (
		sources       []File
		yamlFilesData string
	)

	log.Debug("the files matching the pattern are", slog.Any("pattern", dependency.Path), slog.Any("files-matched", filesMatching))

	for _, fileMatching := range filesMatching {
		yamlFileData, err := os.ReadFile(fileMatching)
		if err != nil {
			return File{}, &errors.YamllError{Message: fmt.Sprintf("reading YAML dependency errored with: '%v'", err)}
		}

		yamlFilesData = yamlFilesData + "\n" + string(yamlFileData)
		sources = append(sources, File{Name: fileMatching, Data: string(yamlFileData)})
	}

	return File{Name: absFilePattern, Data: yamlFilesData, Source: sources}, nil
}

func (dependency *Dependency) FilesFromPattern() ([]File, error) {
	_, filesMatching, err := dependency.filesMatchingPattern()
	if err != nil {
		return nil, err
	}

	files := make([]File, 0, len(filesMatching))

	for _, fileMatching := range filesMatching {
		files = append(files, File{Name: fileMatching})
	}

	return files, nil
}

func (dependency *Dependency) filesMatchingPattern() (string, []string, error) {
	absPatternPath, err := filepath.Abs(filepath.Dir(dependency.Path))
	if err != nil {
		return "", nil, err
	}

	absFilePattern := filepath.Join(absPatternPath, filepath.Base(dependency.Path))

	filesMatching, err := filepath.Glob(absFilePattern)
	if err != nil {
		return "", nil, &errors.YamllError{Message: fmt.Sprintf("error matching pattern: '%v'", err)}
	}

	if len(filesMatching) == 0 {
		return "", nil, &errors.YamllError{Message: fmt.Sprintf("pattern matched no files: '%s'", dependency.Path)}
	}

	var excludePath string
	if dependency.excludePath != "" {
		absExcludePath, err := filepath.Abs(dependency.excludePath)
		if err != nil {
			return "", nil, err
		}

		excludePath = filepath.Clean(absExcludePath)
	}

	filteredFiles := make([]string, 0, len(filesMatching))

	for _, fileMatching := range filesMatching {
		if excludePath != "" && filepath.Clean(fileMatching) == excludePath {
			continue
		}

		filteredFiles = append(filteredFiles, fileMatching)
	}

	return absFilePattern, filteredFiles, nil
}
