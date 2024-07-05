package yamll

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"dario.cat/mergo"
	"github.com/a8m/envsubst"
	"github.com/nikhilsbhat/yamll/pkg/errors"
)

const (
	TypeURL               = "http"
	TypeGit               = "git+"
	TypeFile              = "file"
	defaultDirPermissions = 0o755
)

// Dependency holds the information of the dependencies defined the yaml file.
type Dependency struct {
	Path string `json:"file,omitempty" yaml:"file,omitempty"`
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	Auth Auth   `json:"auth,omitempty" yaml:"auth,omitempty"`
}

// Auth holds the authentication information to resolve the remote yaml files.
type Auth struct {
	// UserName to be used during authenticating remote server.
	UserName string `json:"user_name,omitempty" yaml:"user_name,omitempty"`
	// Password to be used during authenticating remote server.
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	// BarerToken, Define this when you have token and it should be used during authenticating remote server.
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

		yamlFileData, err := dependencyPath.readData(cfg.Effective, cfg.log)
		if err != nil {
			return nil, &errors.YamllError{Message: fmt.Sprintf("reading YAML file errored with: '%v'", err)}
		}

		dependencies, err := cfg.extractDependencies(yamlFileData)
		if err != nil {
			return nil, err
		}

		if fileHierarchy == 0 && !cfg.Root {
			cfg.Root = true
		}

		routes[dependencyPath.Path] = &YamlData{Root: rootFile, File: dependencyPath.Path, DataRaw: yamlFileData, Dependency: dependencies, Index: fileHierarchy}

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

// getDependencyData reads the imports analyses it and generates Dependency data for it.
func (cfg *Config) getDependencyData(dependency string) (*Dependency, error) {
	imports := strings.Split(dependency, ";")
	runeSlice := []rune(imports[0])

	dependencyData := &Dependency{Path: string(runeSlice[4:])}
	dependencyData.identifyType()

	if len(imports) > 1 {
		cfg.log.Debug("auth is set for the import, and implementing the same", slog.String("dependency", dependency))

		authConfig, err := envsubst.String(imports[1])
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

// readData actually reads the data from the identified import.
func (dependency *Dependency) readData(effective bool, log *slog.Logger) (string, error) {
	log.Debug("dependency file type identified", slog.String("type", dependency.Type), slog.Any("path", dependency.Path))

	if effective {
		log.Debug("reading yaml data in effective mode")
	}

	switch {
	case dependency.Type == TypeURL:
		return dependency.URL(log)
	case dependency.Type == TypeGit:
		return dependency.Git(log)
	case dependency.Type == TypeFile:
		return dependency.File(log)
	default:
		return "", &errors.YamllError{Message: fmt.Sprintf("reading data from of type '%s' is not supported", dependency.Type)}
	}
}

// extractDependencies parses the dependencies from YAML file data.
func (cfg *Config) extractDependencies(yamlFileData string) ([]*Dependency, error) {
	var dependencies []*Dependency

	scanner := bufio.NewScanner(strings.NewReader(yamlFileData))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "##++") {
			dependency, err := cfg.getDependencyData(line)
			if err != nil {
				return nil, err
			}

			dependencies = append(dependencies, dependency)
			yamlFileData = strings.ReplaceAll(yamlFileData, line, "")
			yamlFileData = regexp.MustCompile(`[\t\r\n]+`).ReplaceAllString(strings.TrimSpace(yamlFileData), "\n")
		}
	}

	return dependencies, scanner.Err()
}

func (dependency *Dependency) identifyType() {
	switch {
	case strings.HasPrefix(dependency.Path, TypeURL):
		dependency.Type = TypeURL
	case strings.HasPrefix(dependency.Path, TypeGit):
		dependency.Type = TypeGit
	default:
		dependency.Type = TypeFile
	}
}
