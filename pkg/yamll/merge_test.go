package yamll_test

import (
	"testing"

	"github.com/nikhilsbhat/yamll/pkg/yamll"
	"github.com/stretchr/testify/assert"
)

func TestCheckInterDependency(t *testing.T) {
	t.Run("", func(t *testing.T) {
		routes := map[string]*yamll.YamlData{
			"internal/fixtures/import.yaml": {
				File: "internal/fixtures/import.yaml",
				Dependency: []*yamll.Dependency{
					{
						Path: "internal/fixtures/base.yaml",
						Type: "file",
					},
					{
						Path: "internal/fixtures/base2.yaml",
						Type: "file",
					},
				},
			},
			"internal/fixtures/base3.yaml": {
				File: "internal/fixtures/base3.yaml",
				Dependency: []*yamll.Dependency{
					{
						Path: "internal/fixtures/base2.yaml",
						Type: "file",
					},
				},
			},
			"internal/fixtures/base2.yaml": {
				File: "internal/fixtures/base2.yaml",
				Dependency: []*yamll.Dependency{
					{
						Path: "internal/fixtures/base3.yaml",
						Type: "file",
					},
				},
			},
		}

		file := "internal/fixtures/base3.yaml"
		dependency := "internal/fixtures/base2.yaml"

		err := yamll.YamlRoutes(routes).CheckInterDependency(file, dependency)
		assert.NoError(t, err)
	})
}
