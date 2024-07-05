package yamll

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckInterDependency(t *testing.T) {
	t.Run("", func(t *testing.T) {
		routes := map[string]*YamlData{
			"internal/fixtures/import.yaml": {
				File: "internal/fixtures/import.yaml",
				Dependency: []*Dependency{
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
				Dependency: []*Dependency{
					{
						Path: "internal/fixtures/base2.yaml",
						Type: "file",
					},
				},
			},
			"internal/fixtures/base2.yaml": {
				File: "internal/fixtures/base2.yaml",
				Dependency: []*Dependency{
					{
						Path: "internal/fixtures/base3.yaml",
						Type: "file",
					},
				},
			},
		}

		file := "internal/fixtures/base3.yaml"
		dependency := "internal/fixtures/base2.yaml"

		err := YamlRoutes(routes).checkInterDependency(file, dependency)
		assert.NoError(t, err)
	})
}
