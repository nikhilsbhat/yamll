package yamll

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLockEntryFromGitSourcePreservesGitMetadata(t *testing.T) {
	entry := lockEntryFromSource("git+https://example.com/org/repo@main?path=base.yaml", File{
		Name: "base.yaml",
		Data: "name: base\n",
		Meta: FileMeta{
			GitCommit: "deadbeef",
			SHA256:    "abc123",
		},
	})

	require.Equal(t, TypeGit, entry.Type)
	require.Equal(t, "git+https://example.com/org/repo@main?path=base.yaml", entry.Source)
	require.Equal(t, "main", entry.Constraint)
	require.Equal(t, "deadbeef", entry.GitCommit)
	require.Equal(t, "abc123", entry.SHA256)
}

func TestGetGitMetaDataDoesNotMutateDependencyPath(t *testing.T) {
	dependency := &Dependency{Path: "git+https://example.com/org/repo@main?path=base.yaml"}

	meta, err := dependency.getGitMetaData()
	require.NoError(t, err)
	require.Equal(t, "https://example.com/org/repo", meta.gitBaseURL)
	require.Equal(t, "main", meta.referenceName)
	require.Equal(t, "base.yaml", meta.path)
	require.Equal(t, "git+https://example.com/org/repo@main?path=base.yaml", dependency.Path)
}
