package yamll

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"dario.cat/mergo"
	"github.com/a8m/envsubst"
	"github.com/nikhilsbhat/yamll/pkg/errors"
)

const (
	TypeURL               = "http"
	TypeGit               = "git+"
	TypeFile              = "file"
	TypeFilePattern       = "pattern"
	defaultDirPermissions = 0o755
)

// Dependency holds the information of the dependencies defined the yaml file.
type Dependency struct {
	Path        string `json:"file,omitempty" yaml:"file,omitempty"`
	Type        string `json:"type,omitempty" yaml:"type,omitempty"`
	Auth        Auth   `json:"auth,omitempty" yaml:"auth,omitempty"`
	excludePath string
}

// Auth holds the authentication information to resolve the remote yaml files.
type Auth struct {
	// UserName to be used during authenticating remote server.
	UserName string `json:"user_name,omitempty" yaml:"user_name,omitempty"`
	// Password to be used during authenticating remote server.
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	// BarerToken, Define this when you have token, and it should be used during authenticating remote server.
	BarerToken string `json:"barer_token,omitempty" yaml:"barer_token,omitempty"`
	// CaContent, is content of CA bundle if in case you needs to connect to remote server via CA auth.
	CaContent string `json:"ca_content,omitempty" yaml:"ca_content,omitempty"`
	// SSHKey, Path to SSH key to be used while pulling git repository.
	SSHKey string `json:"ssh_key,omitempty" yaml:"ssh_key,omitempty"`
}

// ResolveDependencies addresses the dependencies of YAML imports specified in the YAML files.
func (cfg *Config) ResolveDependencies(routes map[string]*YamlData, dependenciesPath ...*Dependency) (map[string]*YamlData, error) {
	var rootFile bool
	if !cfg.Root {
		rootFile = true
	}

	for fileHierarchy, dependencyPath := range dependenciesPath {
		if _, ok := routes[dependencyPath.Path]; ok {
			continue
		}

		yamlFile, err := dependencyPath.ReadData(cfg.Merge, cfg.log)
		if err != nil {
			return nil, &errors.YamllError{Message: fmt.Sprintf("reading YAML file errored with: '%v'", err)}
		}

		cfg.log.Debug("the absolute path of the file which was read", slog.String("path", yamlFile.Name))

		dependencies, yamlData, err := cfg.extractDependencies(yamlFile.Data, yamlFile.Name)
		if err != nil {
			return nil, err
		}

		if fileHierarchy == 0 && !cfg.Root {
			cfg.Root = true
		}

		sourceFiles := yamlFile.Source
		if len(sourceFiles) == 0 {
			sourceFiles = []File{{Name: yamlFile.Name, Data: yamlFile.Data}}
		}

		routes[dependencyPath.Path] = &YamlData{
			Root:       rootFile,
			File:       dependencyPath.Path,
			DataRaw:    yamlData,
			Dependency: dependencies,
			Index:      fileHierarchy,
			SourceFile: sourceFiles,
		}

		if len(dependencies) != 0 {
			dependencyRoutes, err := cfg.ResolveDependencies(routes, dependencies...)
			if err != nil {
				return nil, err
			}

			if err = mergo.Merge(&routes, dependencyRoutes, mergo.WithOverride); err != nil {
				return nil, &errors.YamllError{Message: fmt.Sprintf("error merging YAML routes: %v", err)}
			}
		}
	}

	return routes, nil
}

// GetDependencyData reads the imports analyses it and generates Dependency data for it.
func (cfg *Config) GetDependencyData(dependency string) (*Dependency, error) {
	importStatement := strings.TrimSpace(dependency)
	if !strings.HasPrefix(importStatement, "##++") {
		return nil, &errors.YamllError{Message: fmt.Sprintf("invalid import statement: %q", dependency)}
	}

	const (
		lengthOfImportWIthAuth = 1
		dependencyImportLength = 2
	)

	imports := strings.SplitN(strings.TrimSpace(strings.TrimPrefix(importStatement, "##++")), ";", dependencyImportLength)

	dependencyPath := strings.TrimSpace(imports[0])
	if dependencyPath == "" {
		return nil, &errors.YamllError{Message: "import path cannot be empty"}
	}

	dependencyData := &Dependency{Path: dependencyPath}
	dependencyData.IdentifyType()

	if len(imports) > lengthOfImportWIthAuth {
		cfg.log.Debug("auth is set for the import, and implementing the same", slog.String("dependency", dependency))

		authConfig, err := envsubst.String(imports[lengthOfImportWIthAuth])
		if err != nil {
			return nil, err
		}

		var auth Auth
		if err := json.Unmarshal([]byte(authConfig), &auth); err != nil {
			return nil, &errors.YamllError{Message: fmt.Sprintf("reading auth config from depency errored with '%v'", err)}
		}

		dependencyData.Auth = auth
	}

	return dependencyData, nil
}

// ReadData actually reads the data from the identified import.
func (dependency *Dependency) ReadData(effective bool, log *slog.Logger) (File, error) {
	log.Debug("dependency file type identified", slog.String("type", dependency.Type), slog.Any("path", dependency.Path))

	if effective {
		log.Debug("reading yaml data in effective mode")
	}

	switch dependency.Type {
	case TypeURL:
		return dependency.URL(log)
	case TypeGit:
		return dependency.Git(log)
	case TypeFile:
		return dependency.File(log)
	case TypeFilePattern:
		return dependency.FilePattern(log)
	default:
		return File{}, &errors.YamllError{Message: fmt.Sprintf("reading data from of type '%s' is not supported", dependency.Type)}
	}
}

// extractDependencies parses the dependencies from YAML file data.
func (cfg *Config) extractDependencies(yamlFileData, sourcePath string) ([]*Dependency, string, error) {
	var (
		dependencies []*Dependency
		cleaned      strings.Builder
	)

	scanner := bufio.NewScanner(strings.NewReader(yamlFileData))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "##++") {
			dependency, err := cfg.GetDependencyData(line)
			if err != nil {
				return nil, "", err
			}

			dependencies = append(dependencies, dependency)

			if dependency.Type == TypeFilePattern && sourcePath != "" {
				dependency.excludePath = sourcePath
			}

			continue
		}

		cleaned.WriteString(line)
		cleaned.WriteByte('\n')
	}

	if err := scanner.Err(); err != nil {
		return nil, "", err
	}

	return dependencies, strings.TrimSpace(cleaned.String()), nil
}

func (dependency *Dependency) IdentifyType() {
	switch {
	case isPattern(dependency.Path):
		dependency.Type = TypeFilePattern
	case strings.HasPrefix(dependency.Path, TypeURL):
		dependency.Type = TypeURL
	case strings.HasPrefix(dependency.Path, TypeGit):
		dependency.Type = TypeGit
	default:
		dependency.Type = TypeFile
	}
}

func isPattern(input string) bool {
	return strings.ContainsAny(input, "*?[]") && !strings.Contains(input, "://")
}
