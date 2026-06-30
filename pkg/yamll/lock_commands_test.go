package yamll_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nikhilsbhat/yamll/pkg/yamll"
	"github.com/stretchr/testify/require"
)

func TestConfigLockVerify(t *testing.T) {
	dir := t.TempDir()
	rootFile := filepath.Join(dir, "root.yaml")
	baseFile := filepath.Join(dir, "base.yaml")
	lockFile := filepath.Join(dir, "yamll.lock")

	require.NoError(t, os.WriteFile(rootFile, []byte("##++"+baseFile+"\napp: true\n"), 0o600))
	require.NoError(t, os.WriteFile(baseFile, []byte("shared: one\n"), 0o600))

	cfg := yamll.New(false, "DEBUG", "", rootFile)
	cfg.SetLogger()
	cfg.LockFile = lockFile

	lockData, err := cfg.Lock()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(lockFile, lockData, 0o600))

	verifyCfg := yamll.New(false, "DEBUG", "", rootFile)
	verifyCfg.SetLogger()
	verifyCfg.LockFile = lockFile
	verifyCfg.NoLock = true

	report, err := verifyCfg.LockVerify()
	require.NoError(t, err)
	require.Equal(t, []string{rootFile}, report.Roots)
	require.Positive(t, report.LockEntriesLoaded)
	require.Positive(t, report.DependenciesResolved)
	require.Contains(t, report.String(), "Lock file is valid")
}

func TestConfigLockExplain(t *testing.T) {
	dir := t.TempDir()
	appRoot := filepath.Join(dir, "app.yaml")
	jobsRoot := filepath.Join(dir, "jobs.yaml")
	sharedFile := filepath.Join(dir, "shared.yaml")
	jobsOnlyFile := filepath.Join(dir, "jobs-only.yaml")

	require.NoError(t, os.WriteFile(appRoot, []byte("##++"+sharedFile+"\napp: true\n"), 0o600))
	require.NoError(t, os.WriteFile(jobsRoot, []byte("##++"+sharedFile+"\n##++"+jobsOnlyFile+"\njobs: true\n"), 0o600))
	require.NoError(t, os.WriteFile(sharedFile, []byte("shared: true\n"), 0o600))
	require.NoError(t, os.WriteFile(jobsOnlyFile, []byte("queue: true\n"), 0o600))

	cfg := yamll.New(false, "DEBUG", "", appRoot, jobsRoot)
	cfg.SetLogger()

	sharedReport, err := cfg.LockExplain(sharedFile)
	require.NoError(t, err)
	require.Equal(t, []string{appRoot, jobsRoot}, sharedReport.Roots)

	jobsReport, err := cfg.LockExplain(jobsOnlyFile)
	require.NoError(t, err)
	require.Equal(t, []string{jobsRoot}, jobsReport.Roots)
	require.Contains(t, jobsReport.String(), "Pulled by roots:")
}
