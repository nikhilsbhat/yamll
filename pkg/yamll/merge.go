package yamll

import "log/slog"

// MergeData combines the YAML file data according to the hierarchy.
func (cfg *Config) MergeData(src string, data map[string]*YamlData) (string, error) {
	for file, fileData := range data {
		if !fileData.Root {
			continue
		}

		out, err := cfg.Merge(src, data, file)
		if err != nil {
			return "", err
		}

		src = out + "\n" + fileData.DataRaw

		cfg.log.Debug("root file was imported successfully", slog.String("file", file))
	}

	return src, nil
}

// Merge actually merges the data when invoked with correct parameters.
func (cfg *Config) Merge(src string, data map[string]*YamlData, file string) (string, error) {
	for _, dependency := range data[file].Dependency {
		if data[dependency.Path].Imported {
			cfg.log.Warn("file already imported hence skipping", slog.String("file", dependency.Path))

			continue
		}

		out, err := cfg.Merge(src, data, dependency.Path)
		if err != nil {
			return "", err
		}

		src = out + "\n"
	}

	if !data[file].Imported && !data[file].Root {
		src = src + "\n" + data[file].DataRaw

		data[file].Imported = true

		cfg.log.Debug("file was imported successfully", slog.String("file", file))
	}

	return src, nil
}
