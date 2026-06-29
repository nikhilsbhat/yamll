package yamll

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/a8m/envsubst"
	"github.com/nikhilsbhat/yamll/pkg/errors"
)

const (
	TypeURL               = "http"
	TypeGit               = "git+"
	TypeOCI               = "oci://"
	TypeFile              = "file"
	TypeFilePattern       = "pattern"
	defaultDirPermissions = 0o755
)

// Dependency holds the information of the dependencies defined the yaml file.
type Dependency struct {
	Path        string `json:"file,omitempty" yaml:"file,omitempty"`
	Type        string `json:"type,omitempty" yaml:"type,omitempty"`
	Auth        *Auth  `json:"auth,omitempty" yaml:"auth,omitempty"`
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

	lockEntries, err := cfg.loadLockEntries()
	if err != nil {
		return nil, err
	}

	for fileHierarchy, dependencyPath := range dependenciesPath {
		if dependencyPath == nil {
			return nil, &errors.YamllError{Message: "dependency path is nil"}
		}

		originalSource := dependencyPath.Path

		if lockEntries != nil && dependencyPath.Type == TypeGit {
			if entry, ok := lockEntries[lockEntryKey(dependencyPath.Path, "")]; ok && entry.GitCommit != "" {
				dependencyPath.Path = pinGitImportToCommit(dependencyPath.Path, entry.GitCommit)
				dependencyPath.IdentifyType()
			}
		}

		if _, ok := routes[dependencyPath.Path]; ok {
			continue
		}

		yamlFile, err := cfg.readDataWithProfile(dependencyPath)
		if err != nil {
			return nil, &errors.YamllError{Message: fmt.Sprintf("reading YAML file errored with: '%v'", err)}
		}

		if err = validateLockedDependency(lockEntries, originalSource, yamlFile); err != nil {
			return nil, err
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
			sourceFiles = []File{{Name: yamlFile.Name, Data: yamlFile.Data, Meta: yamlFile.Meta}}
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

func validateLockedDependency(lockEntries map[string]LockEntry, source string, file File) error {
	if len(lockEntries) == 0 {
		return nil
	}

	if len(file.Source) == 0 {
		return validateSingleLockedFile(lockEntries, source, "", file)
	}

	for _, sourceFile := range file.Source {
		if err := validateSingleLockedFile(lockEntries, source, sourceFile.Name, sourceFile); err != nil {
			return err
		}
	}

	return nil
}

func validateSingleLockedFile(lockEntries map[string]LockEntry, source, patternFile string, file File) error {
	entry, ok := lockEntries[lockEntryKey(source, patternFile)]
	if !ok {
		return &errors.YamllError{Message: fmt.Sprintf("dependency %s is not present in the lock file", source)}
	}

	actual := file.Meta.SHA256
	if actual == "" {
		actual = checksumForContent(file.Data)
	}

	if entry.SHA256 != "" && entry.SHA256 != actual {
		if patternFile == "" {
			return &errors.YamllError{Message: fmt.Sprintf(
				"dependency %s changed since the lock file was generated: expected sha256 %s, got %s",
				source, entry.SHA256, actual,
			)}
		}

		return &errors.YamllError{Message: fmt.Sprintf(
			"pattern dependency %s file %s changed since the lock file was generated: expected sha256 %s, got %s",
			source, patternFile, entry.SHA256, actual,
		)}
	}

	return nil
}

func checksumForContent(content string) string {
	sum := sha256.Sum256([]byte(content))

	return hex.EncodeToString(sum[:])
}

func (cfg *Config) readDataWithProfile(dependencyPath *Dependency) (File, error) {
	readStart := time.Now()

	yamlFile, err := dependencyPath.ReadData(cfg.Merge, cfg.log)
	if err != nil {
		return File{}, err
	}

	if cfg.Profile && cfg.profile != nil && (dependencyPath.Type == TypeURL || dependencyPath.Type == TypeGit) {
		cfg.profile.addRemoteFetch(time.Since(readStart))
	}

	return yamlFile, nil
}

func pinGitImportToCommit(source, commit string) string {
	// git+https://host/org/repo@ref?path=...
	// -> git+https://host/org/repo@<commit>?path=...
	if !strings.HasPrefix(source, TypeGit) {
		return source
	}

	withoutPrefix := strings.TrimPrefix(source, TypeGit)

	beforeAt, afterAt, found := strings.Cut(withoutPrefix, "@")
	if !found {
		return source
	}

	_, afterRef, found := strings.Cut(afterAt, "?")
	if !found {
		return source
	}

	return TypeGit + beforeAt + "@" + commit + "?" + afterRef
}

// PinGitImportToCommitForTest is exported only for tests.
func PinGitImportToCommitForTest(source, commit string) string {
	return pinGitImportToCommit(source, commit)
}

// GetDependencyData reads the imports analyses it and generates Dependency data for it.
func (cfg *Config) GetDependencyData(dependency string) (*Dependency, error) {
	importStatement := strings.TrimSpace(dependency)
	if !strings.HasPrefix(importStatement, "##++") {
		return nil, &errors.YamllError{Message: fmt.Sprintf("invalid import statement: %q", dependency)}
	}

	rawImport := strings.TrimSpace(strings.TrimPrefix(importStatement, "##++"))
	dependencyPath, authPart, hasAuth := strings.Cut(rawImport, ";")
	dependencyPath = strings.TrimSpace(dependencyPath)

	if dependencyPath == "" {
		return nil, &errors.YamllError{Message: "import path cannot be empty"}
	}

	if normalized, ok := normalizeGitShorthand(dependencyPath); ok {
		dependencyPath = normalized
	}

	dependencyData := &Dependency{Path: dependencyPath}
	dependencyData.IdentifyType()

	if hasAuth {
		cfg.log.Debug("auth is set for the import, and implementing the same", slog.String("dependency", dependency))

		authConfig, err := envsubst.String(authPart)
		if err != nil {
			return nil, err
		}

		var auth Auth
		if err := json.Unmarshal([]byte(authConfig), &auth); err != nil {
			return nil, &errors.YamllError{Message: fmt.Sprintf("reading auth config from depency errored with '%v'", err)}
		}

		dependencyData.Auth = &auth
	}

	return dependencyData, nil
}

// normalizeGitShorthand converts a github.com shorthand import like:
// github.com/org/repo//path/to/file.yaml@v1.2.3
// into the canonical git import format.
func normalizeGitShorthand(path string) (string, bool) {
	// Host-agnostic shorthand:
	// <host>/<org>/<repo>//path/to/file.yaml@ref
	// Example: github.com/org/repo//base.yaml@v1.2.0
	// Example: gitlab.com/org/repo//base.yaml@v1.2.0
	//
	// Only https-style hosts are supported by this shorthand.
	if strings.Contains(path, "://") {
		return "", false
	}

	// Split version from suffix.
	base, version, found := strings.Cut(path, "@")
	if !found || version == "" {
		return "", false
	}

	// Split repo from file path.
	repoPart, filePath, found := strings.Cut(base, "//")
	if !found || filePath == "" {
		return "", false
	}

	const splitRootPart = 3

	parts := strings.Split(repoPart, "/")
	if len(parts) < splitRootPart {
		return "", false
	}

	host := parts[0]
	owner := parts[1]
	repo := parts[2]
	gitURL := fmt.Sprintf("https://%s/%s/%s", host, owner, repo)

	return fmt.Sprintf("git+%s@%s?path=%s", gitURL, version, url.QueryEscape(filePath)), true
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
	case TypeOCI:
		return dependency.OCI(log)
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
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "##++") {
			dependency, err := cfg.GetDependencyData(trimmed)
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
	case strings.HasPrefix(dependency.Path, TypeOCI):
		dependency.Type = TypeOCI
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
