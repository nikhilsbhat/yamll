package yamll

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"dario.cat/mergo"
	"github.com/a8m/envsubst"
	"github.com/go-resty/resty/v2"
	"github.com/nikhilsbhat/yamll/pkg/errors"
)

const (
	TypeURL  = "https"
	TypeGit  = "git+"
	TypeFile = "file"
)

// Dependency holds the information of the dependencies defined the yaml file.
type Dependency struct {
	Path string `json:"file,omitempty" yaml:"file,omitempty"`
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	Auth Auth   `json:"auth,omitempty" yaml:"auth,omitempty"`
}

// Auth holds the authentication information to resolve the remote yaml files.
type Auth struct {
	UserName   string `json:"user_name,omitempty" yaml:"user_name,omitempty" mapstructure:"user_name"`
	Password   string `json:"password,omitempty" yaml:"password,omitempty" mapstructure:"password"`
	BarerToken string `json:"barer_token,omitempty" yaml:"barer_token,omitempty" mapstructure:"barer_token"`
	CaContent  string `json:"ca_content,omitempty" yaml:"ca_content,omitempty" mapstructure:"ca_content"`
}

// ResolveDependencies addresses the dependencies of YAML imports specified in the YAML files.
func (cfg *Config) ResolveDependencies(routes map[string]*YamlData, dependenciesPath ...*Dependency) (map[string]*YamlData, error) {
	var rootFile bool
	if !cfg.Root {
		rootFile = true
	}

	for fileHierarchy, dependencyPath := range dependenciesPath {
		yamlFileData, err := dependencyPath.ReadData(cfg.log)
		if err != nil {
			return nil, &errors.YamllError{Message: fmt.Sprintf("reading YAML file errored with: '%v'", err)}
		}

		dependencies := make([]*Dependency, 0)
		stringReader := strings.NewReader(yamlFileData)

		scanner := bufio.NewScanner(stringReader)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "##++") {
				dependency, err := cfg.GetDependencyData(line)
				if err != nil {
					return nil, err
				}

				dependencies = append(dependencies, dependency)
			}
		}

		if fileHierarchy == 0 && !cfg.Root {
			cfg.Root = true
		}

		routes[dependencyPath.Path] = &YamlData{Root: rootFile, File: dependencyPath.Path, DataRaw: yamlFileData, Dependency: dependencies}

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

func (cfg *Config) GetDependencyData(dependency string) (*Dependency, error) {
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

func (dependency *Dependency) ReadData(log *slog.Logger) (string, error) {
	log.Debug("dependency file type identification", slog.String("type", dependency.Type))

	switch {
	case dependency.Type == TypeURL:
		httpClient := resty.New()
		if len(dependency.Auth.BarerToken) != 0 {
			httpClient.SetAuthToken(dependency.Auth.BarerToken)
		}

		if len(dependency.Auth.UserName) != 0 && len(dependency.Auth.Password) != 0 {
			httpClient.SetBasicAuth(dependency.Auth.UserName, dependency.Auth.Password)
		}

		if len(dependency.Auth.CaContent) != 0 {
			certPool := x509.NewCertPool()
			certPool.AppendCertsFromPEM([]byte(dependency.Auth.CaContent))
			httpClient.SetTLSClientConfig(&tls.Config{RootCAs: certPool}) //nolint:gosec
		} else {
			httpClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}) //nolint:gosec
		}

		resp, err := httpClient.R().Get(dependency.Path)
		if err != nil {
			return "", err
		}

		return resp.String(), err
	case dependency.Type == TypeGit:
		return "", &errors.YamllError{Message: "does not support dependency type 'git' at the moment"}
	case dependency.Type == TypeFile:
		absYamlFilePath, err := filepath.Abs(dependency.Path)
		if err != nil {
			return "", err
		}

		yamlFileData, err := os.ReadFile(absYamlFilePath)
		if err != nil {
			return "", &errors.YamllError{Message: fmt.Sprintf("reading YAML dependency errored with: '%v'", err)}
		}

		return string(yamlFileData), nil
	default:
		return "", &errors.YamllError{Message: fmt.Sprintf("reading data from of type '%s' is not supported", dependency.Type)}
	}
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
