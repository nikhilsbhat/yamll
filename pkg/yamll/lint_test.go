package yamll_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nikhilsbhat/yamll/pkg/yamll"
	"github.com/stretchr/testify/require"
)

func TestConfig_Lint(t *testing.T) {
	t.Run("reports unused imports", func(t *testing.T) {
		dir := t.TempDir()

		root := filepath.Join(dir, "root.yaml")
		a := filepath.Join(dir, "a.yaml")
		b := filepath.Join(dir, "b.yaml")

		require.NoError(t, writeFile(root, "##++"+a+"\n##++"+b+"\n\nconfig:\n  <<: *a\n"))
		require.NoError(t, writeFile(a, "a: &a\n  k: v\n"))
		require.NoError(t, writeFile(b, "b: &b\n  k: v\n"))

		cfg := yamll.New(false, "INFO", "---", root)
		cfg.SetLogger()

		report, err := cfg.Lint()
		require.NoError(t, err)

		require.NotEmpty(t, report.Issues)
		require.Contains(t, codes(report.Issues), yamll.LintUnusedImports)
	})

	t.Run("reports invalid anchors", func(t *testing.T) {
		dir := t.TempDir()

		root := filepath.Join(dir, "root.yaml")
		require.NoError(t, writeFile(root, "x: *missing\n"))

		cfg := yamll.New(false, "INFO", "---", root)
		cfg.SetLogger()

		report, err := cfg.Lint()
		require.NoError(t, err)
		require.Contains(t, codes(report.Issues), yamll.LintInvalidAnchors)
	})

	t.Run("reports conflicting merges", func(t *testing.T) {
		dir := t.TempDir()

		root := filepath.Join(dir, "root.yaml")
		lib := filepath.Join(dir, "lib.yaml")

		require.NoError(t, writeFile(root, "##++"+lib+"\n\nobj:\n  <<: *seq\n"))
		require.NoError(t, writeFile(lib, "seq: &seq\n  - one\n  - two\n"))

		cfg := yamll.New(false, "INFO", "---", root)
		cfg.SetLogger()

		report, err := cfg.Lint()
		require.NoError(t, err)
		require.Contains(t, codes(report.Issues), yamll.LintConflictingMerges)
	})

	t.Run("reports duplicate keys", func(t *testing.T) {
		dir := t.TempDir()

		root := filepath.Join(dir, "root.yaml")
		require.NoError(t, writeFile(root, "dup: 1\ndup: 2\n"))

		cfg := yamll.New(false, "INFO", "---", root)
		cfg.SetLogger()

		report, err := cfg.Lint()
		require.NoError(t, err)
		require.Contains(t, codes(report.Issues), yamll.LintDuplicateKeys)
	})

	t.Run("reports circular refs", func(t *testing.T) {
		dir := t.TempDir()

		root := filepath.Join(dir, "root.yaml")
		a := filepath.Join(dir, "a.yaml")
		b := filepath.Join(dir, "b.yaml")

		require.NoError(t, writeFile(root, "##++"+a+"\n\nroot: true\n"))
		require.NoError(t, writeFile(a, "##++"+b+"\n\na: &a\n  k: v\n"))
		require.NoError(t, writeFile(b, "##++"+a+"\n\nb: &b\n  k: v\n"))

		cfg := yamll.New(false, "INFO", "---", root)
		cfg.SetLogger()

		report, err := cfg.Lint()
		require.NoError(t, err)
		require.Contains(t, codes(report.Issues), yamll.LintCircularRefs)
	})
}

func writeFile(path, contents string) error {
	return os.WriteFile(path, []byte(contents), 0o600)
}

func codes(issues []yamll.LintIssue) map[string]struct{} {
	out := make(map[string]struct{}, len(issues))
	for _, i := range issues {
		out[i.Code] = struct{}{}
	}

	return out
}
