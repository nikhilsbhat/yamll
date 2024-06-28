package yamll_test

import (
	"bufio"
	"fmt"
	"strings"
	"testing"

	"github.com/nikhilsbhat/yamll/pkg/yamll"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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
				assert.NoError(t, err)
				dependencies = append(dependencies, dependency)
			}
		}

		for _, dependency := range dependencies {
			fmt.Println(dependency)
		}
	})
}

func TestDependency_ReadData(t *testing.T) {
	t.Run("", func(t *testing.T) {
		dependency := yamll.Dependency{
			Path: "https://run.mocky.io/v3/0a2afb01-5b4a-4bb1-9e5b-5eb7b09330c1",
			Type: yamll.TypeURL,
		}

		cfg := yamll.New(false, "DEBUG", "")
		cfg.SetLogger()

		out, err := dependency.ReadData(false, cfg.GetLogger())
		assert.NoError(t, err)
		fmt.Println(out)
		assert.Nil(t, out)
	})
}

func TestConfig_ResolveDependencies2(t *testing.T) {
	t.Run("", func(t *testing.T) {
		dependency := []*yamll.Dependency{{
			Path: "/path/to/my-opensource/yamll/internal/fixtures/import.yaml",
			Type: yamll.TypeFile,
		}}
		cfg := yamll.New(false, "DEBUG", "/path/to/my-opensource/yamll/internal/fixtures/import.yaml")
		cfg.SetLogger()

		dependencyRoutes, err := cfg.ResolveDependencies(make(map[string]*yamll.YamlData), dependency...)
		assert.NoError(t, err)

		out, err := yaml.Marshal(dependencyRoutes)
		assert.NoError(t, err)
		fmt.Println(string(out))
	})
}

func TestDependency_Git(t *testing.T) {
	t.Run("", func(t *testing.T) {
		cfg := yamll.New(false, "DEBUG", "")
		cfg.SetLogger()

		t.Setenv("GITHUB_TOKEN", "testkey")
		dependency := yamll.Dependency{
			// Path: "git+https://github.com/nikhilsbhat/yamll@main?path=internal/fixtures/base.yaml",
			Path: "git+ssh://git@github.com:nikhilsbhat/yamll@main?path=internal/fixtures/base.yaml",
			Type: "",
			Auth: yamll.Auth{
				UserName: "nikhilsbhat",
				// Password: os.Getenv("GITHUB_TOKEN"),
				SSHKey: "/path/to/ssh/private/key",
			},
		}

		out, err := dependency.Git(cfg.GetLogger())
		assert.NoError(t, err)

		fmt.Println(out)
		assert.Nil(t, out)
	})
}
