package yamll_test

import (
	"strings"
	"testing"

	"github.com/nikhilsbhat/yamll/pkg/yamll"
	"github.com/stretchr/testify/require"
)

func TestYamlExplodePreservesKeyOrder(t *testing.T) {
	inputYAML := yamll.Yaml(strings.Join([]string{
		"default: &default",
		"  apiVersion: v1",
		"  kind: ConfigMap",
		"  metadata:",
		"    name: base-config",
		"  data:",
		"    key1: value1",
		"    key2: value2",
		"config1: *default",
	}, "\n"))

	out, err := inputYAML.Explode()
	require.NoError(t, err)

	lines := strings.Split(string(out), "\n")
	require.Contains(t, lines, "default:")
	require.Contains(t, lines, "config1:")
	require.Less(t, strings.Index(string(out), "default:"), strings.Index(string(out), "config1:"))
	require.Less(t, strings.Index(string(out), "apiVersion: v1"), strings.Index(string(out), "kind: ConfigMap"))
}
