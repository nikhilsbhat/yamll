package yamll

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/nikhilsbhat/yamll/pkg/errors"
)

const lockVersion = "v2"

type LockFile struct {
	Version     string      `yaml:"version"`
	GeneratedAt string      `yaml:"generated_at"`
	Roots       []string    `yaml:"roots"`
	Entries     []LockEntry `yaml:"entries"`
}

type LockEntry struct {
	Type        string `yaml:"type"`
	Source      string `yaml:"source"`
	Constraint  string `yaml:"constraint,omitempty"`
	Resolved    string `yaml:"resolved,omitempty"`
	GitCommit   string `yaml:"git_commit,omitempty"`
	SHA256      string `yaml:"sha256,omitempty"`
	PatternFile string `yaml:"pattern_file,omitempty"`
}

type LockVerifyReport struct {
	Roots                []string
	LockEntriesLoaded    int
	DependenciesResolved int
}

type LockExplainReport struct {
	Target string
	Roots  []string
}

func (cfg *Config) Lock() ([]byte, error) {
	cfg.Root = false

	previousNoLock := cfg.NoLock
	cfg.NoLock = true

	defer func() {
		cfg.NoLock = previousNoLock
	}()

	routes, err := cfg.ResolveDependencies(make(map[string]*YamlData), cfg.Files...)
	if err != nil {
		return nil, &errors.YamllError{Message: fmt.Sprintf("fetching dependency tree errored with: '%v'", err)}
	}

	var entries []LockEntry

	for _, file := range YamlRoutes(routes).OrderedFiles() {
		route := routes[file]
		for _, src := range route.SourceFile {
			entries = append(entries, lockEntryFromSource(route.File, src))
		}
	}

	// Deterministic ordering.
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Source != entries[j].Source {
			return entries[i].Source < entries[j].Source
		}

		return entries[i].PatternFile < entries[j].PatternFile
	})

	lock := LockFile{
		Version:     lockVersion,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Roots:       cfg.rootPaths(),
		Entries:     entries,
	}

	out, err := yaml.MarshalWithOptions(lock, yaml.Indent(yamlIndent), yaml.IndentSequence(true))
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (cfg *Config) LockVerify() (LockVerifyReport, error) {
	if cfg.LockFile == "" {
		return LockVerifyReport{}, &errors.YamllError{Message: "lock file path cannot be empty"}
	}

	if _, err := os.Stat(cfg.LockFile); err != nil {
		return LockVerifyReport{}, err
	}

	previousNoLock := cfg.NoLock
	cfg.NoLock = false

	defer func() {
		cfg.NoLock = previousNoLock
	}()

	lockEntries, err := cfg.loadLockEntries()
	if err != nil {
		return LockVerifyReport{}, err
	}

	cfg.Root = false

	routes, err := cfg.ResolveDependencies(make(map[string]*YamlData), cfg.Files...)
	if err != nil {
		return LockVerifyReport{}, &errors.YamllError{Message: fmt.Sprintf("verifying lock file errored with: '%v'", err)}
	}

	return LockVerifyReport{
		Roots:                cfg.rootPaths(),
		LockEntriesLoaded:    len(lockEntries),
		DependenciesResolved: len(routes),
	}, nil
}

func (cfg *Config) LockExplain(target string) (LockExplainReport, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return LockExplainReport{}, &errors.YamllError{Message: "lock explain requires a dependency source"}
	}

	roots := make([]string, 0, len(cfg.Files))

	for _, root := range cfg.Files {
		if root == nil || root.Path == "" {
			continue
		}

		routes, err := cfg.resolveSingleRootWithoutLock(root)
		if err != nil {
			return LockExplainReport{}, err
		}

		if dependencyTreeContainsTarget(routes, target) {
			roots = append(roots, root.Path)
		}
	}

	sort.Strings(roots)

	return LockExplainReport{Target: target, Roots: roots}, nil
}

func (cfg *Config) resolveSingleRootWithoutLock(root *Dependency) (map[string]*YamlData, error) {
	rootCopy := *root
	resolveCfg := *cfg
	resolveCfg.Root = false
	resolveCfg.NoLock = true
	resolveCfg.Files = []*Dependency{&rootCopy}

	routes, err := resolveCfg.ResolveDependencies(make(map[string]*YamlData), &rootCopy)
	if err != nil {
		return nil, &errors.YamllError{Message: fmt.Sprintf("resolving root %s errored with: '%v'", root.Path, err)}
	}

	return routes, nil
}

func dependencyTreeContainsTarget(routes map[string]*YamlData, target string) bool {
	for file, route := range routes {
		if lockPathMatches(file, target) {
			return true
		}

		if route == nil {
			continue
		}

		for _, sourceFile := range route.SourceFile {
			if lockPathMatches(sourceFile.Name, target) {
				return true
			}
		}
	}

	return false
}

func lockPathMatches(path, target string) bool {
	if path == target {
		return true
	}

	if strings.Contains(path, "://") || strings.Contains(target, "://") {
		return false
	}

	return filepath.Clean(path) == filepath.Clean(target)
}

func (r LockVerifyReport) String() string {
	return fmt.Sprintf(
		"Lock file is valid\nRoots checked: %d\nLock entries loaded: %d\nDependencies resolved: %d\n",
		len(r.Roots),
		r.LockEntriesLoaded,
		r.DependenciesResolved,
	)
}

func (r LockExplainReport) String() string {
	lines := []string{"Dependency: " + r.Target, "Pulled by roots:"}

	if len(r.Roots) == 0 {
		lines = append(lines, "  <none>")
	} else {
		for _, root := range r.Roots {
			lines = append(lines, "  "+root)
		}
	}

	lines = append(lines, "", fmt.Sprintf("Total roots: %d", len(r.Roots)))

	return strings.Join(lines, "\n") + "\n"
}

func (cfg *Config) rootPaths() []string {
	roots := make([]string, 0, len(cfg.Files))

	for _, dep := range cfg.Files {
		if dep != nil && dep.Path != "" {
			roots = append(roots, dep.Path)
		}
	}

	return roots
}

func lockEntryFromSource(source string, file File) LockEntry {
	entry := LockEntry{
		Source: source,
		SHA256: file.Meta.SHA256,
	}

	switch {
	case file.Meta.GitCommit != "":
		entry.Type = TypeGit
		entry.Constraint = gitConstraintFromSource(source)
		entry.GitCommit = file.Meta.GitCommit
		entry.Resolved = file.Name
	case isPattern(source):
		entry.Type = TypeFilePattern
		entry.Resolved = file.Name
		entry.PatternFile = file.Name
	default:
		// Heuristic: Name is URL if it begins with http(s).
		if len(file.Name) >= 4 && (file.Name[:4] == "http") {
			entry.Type = TypeURL
			entry.Resolved = file.Name
		} else {
			entry.Type = TypeFile
			entry.Resolved = file.Name
		}
	}

	if entry.SHA256 == "" {
		sum := sha256.Sum256([]byte(file.Data))
		entry.SHA256 = hex.EncodeToString(sum[:])
	}

	return entry
}

func gitConstraintFromSource(source string) string {
	// Extract ref from git import when possible:
	// git+https://host/org/repo@ref?path=...
	if !strings.HasPrefix(source, TypeGit) {
		return ""
	}

	withoutPrefix := strings.TrimPrefix(source, TypeGit)

	_, afterAt, found := strings.Cut(withoutPrefix, "@")
	if !found {
		return ""
	}

	ref, _, _ := strings.Cut(afterAt, "?")

	return ref
}

func lockEntryKey(source, patternFile string) string {
	if patternFile == "" {
		return source
	}

	return source + "\x00" + patternFile
}
