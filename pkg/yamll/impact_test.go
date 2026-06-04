package yamll_test

import (
	"path/filepath"
	"testing"

	"github.com/nikhilsbhat/yamll/pkg/yamll"
	"github.com/stretchr/testify/require"
)

func TestConfig_Impact(t *testing.T) {
	dir := t.TempDir()

	common := filepath.Join(dir, "common.yaml")
	api := filepath.Join(dir, "api.yaml")
	ingress := filepath.Join(dir, "ingress.yaml")
	web := filepath.Join(dir, "web.yaml")
	jobs := filepath.Join(dir, "jobs.yaml")
	root := filepath.Join(dir, "root.yaml")

	require.NoError(t, writeFile(common, "service:\n  port: 8080\n"))
	require.NoError(t, writeFile(api, "##++"+common+"\napi:\n  enabled: true\n"))
	require.NoError(t, writeFile(ingress, "##++"+common+"\ningress:\n  enabled: true\n"))
	require.NoError(t, writeFile(web, "##++"+api+"\n##++"+ingress+"\nweb:\n  enabled: true\n"))
	require.NoError(t, writeFile(jobs, "##++"+common+"\njobs:\n  enabled: true\n"))
	require.NoError(t, writeFile(root, "##++"+web+"\n##++"+jobs+"\nroot: true\n"))

	cfg := yamll.New(false, "INFO", "---", root)
	cfg.SetLogger()

	report, err := cfg.Impact(common)
	require.NoError(t, err)
	require.Equal(t, common, report.Target)
	require.Len(t, report.Affected, 4)
	require.Contains(t, report.Affected, api)
	require.Contains(t, report.Affected, ingress)
	require.Contains(t, report.Affected, web)
	require.Contains(t, report.Affected, jobs)
	require.Equal(t, 4, report.Total)
}
