package yamll_test

import (
	"testing"

	"github.com/nikhilsbhat/yamll/pkg/yamll"
	"github.com/stretchr/testify/assert"
)

func Test_fetchDependency(t *testing.T) {
	t.Run("", func(t *testing.T) {
		cfg := yamll.New("yamll/internal/fixtures/import.yaml", "")
		deps, err := cfg.Yaml()
		assert.NoError(t, err)

		assert.NotNil(t, deps)
	})
}
