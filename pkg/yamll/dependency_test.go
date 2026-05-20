package yamll_test

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nikhilsbhat/yamll/pkg/yamll"
	"github.com/stretchr/testify/require"
)

func Test_getDependencyData2(t *testing.T) {
	t.Run("", func(t *testing.T) {
		absYamlFilePath := `##++internal/fixtures/base.yaml
##++internal/fixtures/base2.yaml
##++path/to/test.yaml
##++git+https://github.com/nikhilsbhat/yamll/blob/main/internal/fixtures/base.yaml
##++https://test.com/test.yaml;{"user_name":"${USERNAME}","password":"${PASSWORD}","ca_content":"${CA_CONTENT}"}
##++https://run.mocky.io/v3/92e08b25-dd1f-4dd0-bc55-9649b5b896c9`

		stringReader := strings.NewReader(absYamlFilePath)

		scanner := bufio.NewScanner(stringReader)

		t.Setenv("USERNAME", "nikhil")

		t.Setenv("PASSWORD", "super-secret-password")

		cfg := yamll.New(false, "DEBUG", "")

		cfg.SetLogger()

		dependencies := make([]*yamll.Dependency, 0)

		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "##++") {
				dependency, err := cfg.GetDependencyData(line)
				require.NoError(t, err)

				dependencies = append(dependencies, dependency)
			}
		}

		require.Len(t, dependencies, 6)
		require.Equal(t, "https://test.com/test.yaml", dependencies[4].Path)
		require.Equal(t, "nikhil", dependencies[4].Auth.UserName)
		require.Equal(t, "super-secret-password", dependencies[4].Auth.Password)
	})
}

func TestDependency_ReadData(t *testing.T) {
	t.Run("url success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("name: yamll\n"))
		}))
		defer server.Close()

		dependency := yamll.Dependency{
			Path: server.URL,
			Type: yamll.TypeURL,
		}

		cfg := yamll.New(false, "DEBUG", "")

		cfg.SetLogger()

		out, err := dependency.ReadData(false, cfg.GetLogger())
		require.NoError(t, err)
		require.Equal(t, server.URL, out.Name)
		require.Equal(t, "name: yamll", out.Data)
	})

	t.Run("url error status", func(t *testing.T) {
		server := httptest.NewServer(http.NotFoundHandler())
		defer server.Close()

		dependency := yamll.Dependency{
			Path: server.URL,
			Type: yamll.TypeURL,
		}

		cfg := yamll.New(false, "DEBUG", "")
		cfg.SetLogger()

		_, err := dependency.ReadData(false, cfg.GetLogger())
		require.Error(t, err)
	})
}

func TestConfig_ResolveDependencies2(t *testing.T) {
	t.Run("", func(t *testing.T) {
		wd, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir("../.."))
		t.Cleanup(func() {
			require.NoError(t, os.Chdir(wd))
		})

		dependency := []*yamll.Dependency{{
			Path: "internal/fixtures/base.yaml",
			Type: yamll.TypeFile,
		}}
		cfg := yamll.New(false, "DEBUG", "internal/fixtures/base.yaml")
		cfg.SetLogger()

		dependencyRoutes, err := cfg.ResolveDependencies(make(map[string]*yamll.YamlData), dependency...)
		require.NoError(t, err)
		require.NotContains(t, dependencyRoutes["internal/fixtures/base.yaml"].DataRaw, "##++")
	})
}

func TestDependency_Git(t *testing.T) {
	t.Run("identifies git import", func(t *testing.T) {
		dependency := yamll.Dependency{
			Path: "git+https://github.com/nikhilsbhat/yamll@main?path=internal/fixtures/base.yaml",
		}

		dependency.IdentifyType()

		require.Equal(t, yamll.TypeGit, dependency.Type)
	})
}

func TestConfig_YamlWithFilePattern(t *testing.T) {
	dir := t.TempDir()
	rootFile := filepath.Join(dir, "root.yaml")
	pattern := filepath.Join(dir, "*.yaml")

	require.NoError(t, os.WriteFile(rootFile, []byte("##++"+pattern+"\nroot: true\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "one.yaml"), []byte("one: 1\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "two.yaml"), []byte("two: 2\n"), 0o600))

	cfg := yamll.New(false, "DEBUG", "---", rootFile)
	cfg.SetLogger()

	out, err := cfg.Yaml()
	require.NoError(t, err)
	require.Contains(t, string(out), "one: 1")
	require.Contains(t, string(out), "two: 2")
	require.Contains(t, string(out), "root: true")
	require.Equal(t, 1, strings.Count(string(out), "root: true"))

	repeatedOut, err := cfg.Yaml()
	require.NoError(t, err)
	require.Equal(t, out, repeatedOut)

	for range 10 {
		cfg := yamll.New(false, "DEBUG", "---", rootFile)
		cfg.SetLogger()

		nextOut, err := cfg.Yaml()
		require.NoError(t, err)
		require.Equal(t, out, nextOut)
	}
}

func TestConfig_YamlBuildWithPatternKeepsFirstAnchor(t *testing.T) {
	dir := t.TempDir()
	rootFile := filepath.Join(dir, "root.yaml")

	require.NoError(t, os.WriteFile(rootFile, []byte(strings.Join([]string{
		"##++" + filepath.Join(dir, "*.test.yaml"),
		"##++" + filepath.Join(dir, "*.testing.yaml"),
		"base:",
		"  <<: *base_test",
		"  <<: *base2_test",
		"  <<: *base3_test",
		"",
	}, "\n")), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "base.test.yaml"), []byte(
		"##++"+filepath.Join(dir, "editor-map.yaml")+"\nbase_test: &base_test\n  base_name: base_test\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "base2.test.yaml"), []byte(
		"##++"+filepath.Join(dir, "movies-map.yaml")+"\nbase2_test: &base2_test\n  base2_name: base2_test\n  <<: *movies\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "base3.test.yaml"), []byte(
		"##++"+filepath.Join(dir, "editor-map.yaml")+"\nbase3_test: &base3_test\n  base3_name: base3_test\n  <<: *editor\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "editor-map.yaml"), []byte(
		"editor: &editor\n  editor:\n    - intellij\n    - visual_code\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "movies-map.yaml"), []byte(
		"movies: &movies\n  movies:\n    - animation\n    - comedy\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "one.testing.yaml"), []byte(
		"editor: &editor\n  - intellij\n  - visual_code\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "two.testing.yaml"), []byte(
		"movies: &movies\n  - animation\n  - comedy\n"), 0o600))

	cfg := yamll.New(false, "DEBUG", "---", rootFile)
	cfg.SetLogger()

	out, err := cfg.YamlBuild()
	require.NoError(t, err)
	require.Contains(t, string(out), "movies:")
	require.Contains(t, string(out), "- animation")
	require.Contains(t, string(out), "editor:")
	require.Contains(t, string(out), "- intellij")
	require.Less(t, strings.Index(string(out), "base_name:"), strings.Index(string(out), "base2_name:"))
	require.Less(t, strings.Index(string(out), "base2_name:"), strings.Index(string(out), "base3_name:"))
}

func TestConfig_YamlBuildKeepsRootKeyOrder(t *testing.T) {
	dir := t.TempDir()
	rootFile := filepath.Join(dir, "root.yaml")

	require.NoError(t, os.WriteFile(rootFile, []byte(strings.Join([]string{
		"third:",
		"  value: 3",
		"first:",
		"  value: 1",
		"second:",
		"  value: 2",
		"",
	}, "\n")), 0o600))

	cfg := yamll.New(false, "DEBUG", "---", rootFile)
	cfg.SetLogger()

	out, err := cfg.YamlBuild()
	require.NoError(t, err)
	require.Less(t, strings.Index(string(out), "third:"), strings.Index(string(out), "first:"))
	require.Less(t, strings.Index(string(out), "first:"), strings.Index(string(out), "second:"))
}

func TestConfig_TraceFindsDirectAndMergedOrigins(t *testing.T) {
	dir := t.TempDir()
	rootFile := filepath.Join(dir, "root.yaml")
	baseFile := filepath.Join(dir, "base.yaml")

	require.NoError(t, os.WriteFile(rootFile, []byte(strings.Join([]string{
		"##++" + baseFile,
		"service:",
		"  <<: *service_base",
		"  name: api",
		"",
	}, "\n")), 0o600))
	require.NoError(t, os.WriteFile(baseFile, []byte(strings.Join([]string{
		"service_base: &service_base",
		"  metadata:",
		"    labels:",
		"      app: api",
		"",
	}, "\n")), 0o600))

	cfg := yamll.New(false, "DEBUG", "---", rootFile)
	cfg.SetLogger()

	directOrigin, err := cfg.Trace("service.name")
	require.NoError(t, err)
	require.Equal(t, displayTestPath(rootFile)+":4", directOrigin.Origin)

	mergedOrigin, err := cfg.Trace("service.metadata.labels")
	require.NoError(t, err)
	require.Equal(t, displayTestPath(baseFile)+":3", mergedOrigin.Origin)
}

func TestConfig_TraceFindsPatternFileOrigin(t *testing.T) {
	dir := t.TempDir()
	rootFile := filepath.Join(dir, "root.yaml")
	pattern := filepath.Join(dir, "*.test.yaml")
	baseFile := filepath.Join(dir, "base.test.yaml")

	require.NoError(t, os.WriteFile(rootFile, []byte(strings.Join([]string{
		"##++" + pattern,
		"base:",
		"  <<: *base_test",
		"",
	}, "\n")), 0o600))
	require.NoError(t, os.WriteFile(baseFile, []byte(strings.Join([]string{
		"base_test: &base_test",
		"  base_name: base_test",
		"",
	}, "\n")), 0o600))

	cfg := yamll.New(false, "DEBUG", "---", rootFile)
	cfg.SetLogger()

	origin, err := cfg.Trace("base.base_name")
	require.NoError(t, err)
	require.Equal(t, displayTestPath(baseFile)+":2", origin.Origin)
}

func displayTestPath(path string) string {
	relPath, err := filepath.Rel(".", path)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return path
	}

	return relPath
}
