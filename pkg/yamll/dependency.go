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
	"regexp"
	"strings"

	"dario.cat/mergo"
	"github.com/a8m/envsubst"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/nikhilsbhat/yamll/pkg/errors"
	"golang.org/x/crypto/ssh"
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

type gitMeta struct {
	gitBaseURL    string
	referenceName string
	path          string
	ssh           bool
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
				yamlFileData = strings.ReplaceAll(yamlFileData, line, "")
				yamlFileData = regexp.MustCompile(`[\t\r\n]+`).ReplaceAllString(strings.TrimSpace(yamlFileData), "\n")
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

// GetDependencyData reads the imports analyses it and generates Dependency data for it.
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

// ReadData actually reads the data from the identified import.
func (dependency *Dependency) ReadData(log *slog.Logger) (string, error) {
	log.Debug("dependency file type identified", slog.String("type", dependency.Type), slog.Any("path", dependency.Path))

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

// Git reads the data from the Git import.
func (dependency *Dependency) Git(log *slog.Logger) (string, error) {
	gitMetaData, err := dependency.getGitMetaData()
	if err != nil {
		return "", err
	}

	cloneOptions := &git.CloneOptions{
		URL:      gitMetaData.gitBaseURL,
		Progress: os.Stdout,
	}

	if len(dependency.Auth.CaContent) != 0 {
		cloneOptions.CABundle = []byte(dependency.Auth.CaContent)
	}

	switch gitMetaData.ssh {
	case true:
		log.Debug("the git import is of type ssh, so setting ssh based auth")

		sshKEY, err := os.ReadFile(dependency.Auth.SSHKey)
		if err != nil {
			return "", &errors.YamllError{Message: fmt.Sprintf("reading ssh key '%s' errored with %v", dependency.Auth.SSHKey, err)}
		}

		signer, err := ssh.ParsePrivateKey(sshKEY)
		if err != nil {
			return "", err
		}

		cloneOptions.Auth = &gitssh.PublicKeys{User: "git", Signer: signer}

	case false:
		log.Debug("the git import is of type https, so setting http based auth")

		auth := &http.BasicAuth{
			Username: dependency.Auth.UserName,
			Password: dependency.Auth.Password,
		}

		if len(dependency.Auth.BarerToken) != 0 {
			auth.Password = dependency.Auth.BarerToken
		}

		cloneOptions.Auth = auth
	}

	tempDir := filepath.Join(os.TempDir(), "yamll_git"+uuid.New().String())
	if err = os.MkdirAll(tempDir, defaultDirPermissions); err != nil {
		return "", &errors.YamllError{Message: "failed to crete temp directory for cloning git material"}
	}

	log.Debug("cloning git repo", slog.String("repo", gitMetaData.gitBaseURL), slog.String("dir", tempDir))

	defer func(path string) {
		if err = os.RemoveAll(path); err != nil {
			log.Error(err.Error())
		}
	}(tempDir)

	repo, err := git.PlainClone(tempDir, false, cloneOptions)
	if err != nil {
		return "", err
	}

	if err = checkoutRevision(repo, gitMetaData.referenceName); err != nil {
		return "", err
	}

	gitFileContent, err := os.ReadFile(filepath.Join(tempDir, gitMetaData.path))
	if err != nil {
		return "", &errors.YamllError{Message: fmt.Sprintf("reading content from file of git errored with '%v'", err)}
	}

	return string(gitFileContent), nil
}

// URL reads the data from the URL import.
func (dependency *Dependency) URL(log *slog.Logger) (string, error) {
	httpClient := resty.New()

	if len(dependency.Auth.BarerToken) != 0 {
		log.Debug("using token based auth for remote URL", slog.Any("url", dependency.Path))

		httpClient.SetAuthToken(dependency.Auth.BarerToken)
	}

	if len(dependency.Auth.UserName) != 0 && len(dependency.Auth.Password) != 0 {
		log.Debug("using basic auth for remote URL", slog.Any("url", dependency.Path))

		httpClient.SetBasicAuth(dependency.Auth.UserName, dependency.Auth.Password)
	}

	if len(dependency.Auth.CaContent) != 0 {
		log.Debug("using CA for authentication for remote URL", slog.Any("url", dependency.Path))

		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM([]byte(dependency.Auth.CaContent))
		httpClient.SetTLSClientConfig(&tls.Config{RootCAs: certPool}) //nolint:gosec
	} else {
		log.Debug("skipping TLS verification")

		httpClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}) //nolint:gosec
	}

	resp, err := httpClient.R().Get(dependency.Path)
	if err != nil {
		return "", err
	}

	return resp.String(), err
}

// File reads the data from the File import.
func (dependency *Dependency) File(_ *slog.Logger) (string, error) {
	absYamlFilePath, err := filepath.Abs(dependency.Path)
	if err != nil {
		return "", err
	}

	yamlFileData, err := os.ReadFile(absYamlFilePath)
	if err != nil {
		return "", &errors.YamllError{Message: fmt.Sprintf("reading YAML dependency errored with: '%v'", err)}
	}

	return string(yamlFileData), nil
}

//nolint:gomnd
func (dependency *Dependency) getGitMetaData() (*gitMeta, error) {
	dependency.Path = strings.ReplaceAll(dependency.Path, "git+", "")

	var isSSH bool

	var gitBaseURL string

	if strings.HasPrefix(dependency.Path, "ssh://") {
		isSSH = true

		gitParsedURL := strings.SplitN(dependency.Path, "@", 3)
		if len(gitParsedURL) != 3 {
			return nil, &errors.YamllError{Message: fmt.Sprintf("unable to split git url '%s'", dependency.Path)}
		}

		gitBaseURL = fmt.Sprintf("git@%v", gitParsedURL[1])
		dependency.Path = fmt.Sprintf("https://%v@%v", gitParsedURL[1], gitParsedURL[2])
	} else {
		gitParsedURL := strings.SplitN(dependency.Path, "@", 2)
		if len(gitParsedURL) != 2 {
			return nil, &errors.YamllError{Message: fmt.Sprintf("unable to parse git url '%s'", dependency.Path)}
		}
		gitBaseURL = gitParsedURL[0]
	}

	parsedRef := strings.SplitN(strings.SplitN(dependency.Path, "?", 2)[0], "@", 2)
	if len(parsedRef) != 2 {
		return nil, &errors.YamllError{Message: fmt.Sprintf("unable to parse ref from '%s'", dependency.Path)}
	}

	parsedPath := strings.SplitN(strings.SplitN(dependency.Path, "?", 2)[1], "=", 2)
	if len(parsedPath) != 2 {
		return nil, &errors.YamllError{Message: fmt.Sprintf("unable to parse path from '%s'", dependency.Path)}
	}

	return &gitMeta{
		gitBaseURL:    gitBaseURL,
		referenceName: parsedRef[1],
		path:          parsedPath[1],
		ssh:           isSSH,
	}, nil
}

func checkoutRevision(repo *git.Repository, referenceName string) error {
	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	hash, err := repo.ResolveRevision(plumbing.Revision(referenceName))
	if err != nil {
		return err
	}

	return worktree.Checkout(&git.CheckoutOptions{Hash: *hash})
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
