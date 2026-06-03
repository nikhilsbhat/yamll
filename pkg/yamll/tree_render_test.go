package yamll_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nikhilsbhat/yamll/pkg/yamll"
	"github.com/stretchr/testify/require"
)

func TestConfig_TreeFormats(t *testing.T) {
	dir := t.TempDir()

	root := filepath.Join(dir, "root.yaml")
	base := filepath.Join(dir, "base.yaml")
	patternOne := filepath.Join(dir, "one.test.yaml")
	patternTwo := filepath.Join(dir, "two.test.yaml")

	require.NoError(t, writeFile(root, strings.Join([]string{
		"##++" + base,
		"##++" + filepath.Join(dir, "*.test.yaml"),
		"",
		"root: true",
	}, "\n")))
	require.NoError(t, writeFile(base, "base: &base\n  enabled: true\n"))
	require.NoError(t, writeFile(patternOne, "one: &one\n  value: one\n"))
	require.NoError(t, writeFile(patternTwo, "two: &two\n  value: two\n"))

	cfg := yamll.New(false, "INFO", "---", root)
	cfg.SetLogger()

	t.Run("text", func(t *testing.T) {
		out, err := cfg.Tree(yamll.TreeOutputText, true, true)
		require.NoError(t, err)
		require.Contains(t, out, root)
		require.Contains(t, out, base)
		require.Contains(t, out, "*.test.yaml (2 files)")
	})

	t.Run("json", func(t *testing.T) {
		out, err := cfg.Tree(yamll.TreeOutputJSON, true, true)
		require.NoError(t, err)

		var node yamll.DependencyTreeNode

		require.NoError(t, json.Unmarshal([]byte(out), &node))
		require.Equal(t, root, node.Name)
		require.Len(t, node.Children, 2)
	})

	t.Run("dot", func(t *testing.T) {
		out, err := cfg.Tree(yamll.TreeOutputDOT, true, true)
		require.NoError(t, err)
		require.Contains(t, out, "digraph yamll")
		require.Contains(t, out, "TestConfig_TreeFormats")
		require.Contains(t, out, "001/base.yaml")
		require.NotContains(t, out, "/var/")
	})

	t.Run("mermaid", func(t *testing.T) {
		out, err := cfg.Tree(yamll.TreeOutputMermaid, true, true)
		require.NoError(t, err)
		require.Contains(t, out, "graph TD")
		require.Contains(t, out, "<br/>")
		require.NotContains(t, out, "/var/")
	})

	t.Run("invalid format", func(t *testing.T) {
		_, err := cfg.Tree("xml", true, true)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported tree output format")
	})
}
