package yamll_test

import (
	"testing"

	"github.com/nikhilsbhat/yamll/pkg/yamll"
	"github.com/stretchr/testify/require"
)

func Test_fetchDependency(t *testing.T) {
	t.Run("", func(t *testing.T) {
		cfg := yamll.New(false, "yamll/internal/fixtures/import.yaml", "")
		deps, err := cfg.Yaml()
		require.NoError(t, err)

		require.NotNil(t, deps)
	})
}
