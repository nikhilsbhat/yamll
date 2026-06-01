package yamll

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

func (cfg *Config) Lock() ([]byte, error) {
	cfg.Root = false

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
