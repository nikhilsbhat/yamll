package yamll_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nikhilsbhat/yamll/pkg/yamll"
	"github.com/stretchr/testify/require"
)

func TestConfigYamlFailsWhenLockedFileContentChanges(t *testing.T) {
	dir := t.TempDir()
	rootFile := filepath.Join(dir, "root.yaml")
	baseFile := filepath.Join(dir, "base.yaml")
	lockFile := filepath.Join(dir, "yamll.lock")

	require.NoError(t, os.WriteFile(rootFile, []byte("##++"+baseFile+"\napp: true\n"), 0o600))
	require.NoError(t, os.WriteFile(baseFile, []byte("shared: one\n"), 0o600))

	generateCfg := yamll.New(false, "DEBUG", "", rootFile)
	generateCfg.SetLogger()
	generateCfg.LockFile = lockFile

	lockData, err := generateCfg.Lock()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(lockFile, lockData, 0o600))

	require.NoError(t, os.WriteFile(baseFile, []byte("shared: two\n"), 0o600))

	runCfg := yamll.New(false, "DEBUG", "", rootFile)
	runCfg.SetLogger()
	runCfg.LockFile = lockFile

	_, err = runCfg.Yaml()
	require.Error(t, err)
	require.Contains(t, err.Error(), "changed since the lock file was generated")
	require.Contains(t, err.Error(), "dependency "+baseFile)
	require.NotContains(t, err.Error(), `\"`)
}

func TestConfigYamlFailsWhenLockedPatternFileContentChanges(t *testing.T) {
	dir := t.TempDir()
	rootFile := filepath.Join(dir, "root.yaml")
	lockFile := filepath.Join(dir, "yamll.lock")
	pattern := filepath.Join(dir, "*.yaml")
	firstFile := filepath.Join(dir, "one.yaml")
	secondFile := filepath.Join(dir, "two.yaml")

	require.NoError(t, os.WriteFile(rootFile, []byte("##++"+pattern+"\napp: true\n"), 0o600))
	require.NoError(t, os.WriteFile(firstFile, []byte("first: one\n"), 0o600))
	require.NoError(t, os.WriteFile(secondFile, []byte("second: two\n"), 0o600))

	generateCfg := yamll.New(false, "DEBUG", "", rootFile)
	generateCfg.SetLogger()
	generateCfg.LockFile = lockFile

	lockData, err := generateCfg.Lock()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(lockFile, lockData, 0o600))

	require.NoError(t, os.WriteFile(secondFile, []byte("second: changed\n"), 0o600))

	runCfg := yamll.New(false, "DEBUG", "", rootFile)
	runCfg.SetLogger()
	runCfg.LockFile = lockFile

	_, err = runCfg.Yaml()
	require.Error(t, err)
	require.Contains(t, err.Error(), "changed since the lock file was generated")
	require.Contains(t, err.Error(), "pattern dependency "+pattern)
	require.Contains(t, err.Error(), "file "+secondFile)
	require.NotContains(t, err.Error(), `\"`)
}

func TestConfigLockRegeneratesWhenExistingLockIsStale(t *testing.T) {
	dir := t.TempDir()
	rootFile := filepath.Join(dir, "root.yaml")
	baseFile := filepath.Join(dir, "base.yaml")
	lockFile := filepath.Join(dir, "yamll.lock")

	require.NoError(t, os.WriteFile(rootFile, []byte("##++"+baseFile+"\napp: true\n"), 0o600))
	require.NoError(t, os.WriteFile(baseFile, []byte("shared: one\n"), 0o600))

	generateCfg := yamll.New(false, "DEBUG", "", rootFile)
	generateCfg.SetLogger()
	generateCfg.LockFile = lockFile

	lockData, err := generateCfg.Lock()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(lockFile, lockData, 0o600))

	require.NoError(t, os.WriteFile(baseFile, []byte("shared: two\n"), 0o600))

	regenerateCfg := yamll.New(false, "DEBUG", "", rootFile)
	regenerateCfg.SetLogger()
	regenerateCfg.LockFile = lockFile

	nextLockData, err := regenerateCfg.Lock()
	require.NoError(t, err)
	require.Contains(t, string(nextLockData), "sha256:")
}
