package yamll

import (
	"errors"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	pkgErrors "github.com/nikhilsbhat/yamll/pkg/errors"
)

// nolint:nilnil
func (cfg *Config) loadLockEntries() (map[string]LockEntry, error) {
	if cfg.NoLock || cfg.LockFile == "" {
		return nil, nil
	}

	data, err := os.ReadFile(cfg.LockFile)
	if err != nil {
		// Lock file is optional unless user runs `yamll lock`.
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, err
	}

	var lock LockFile

	if err = yaml.Unmarshal(data, &lock); err != nil {
		return nil, &pkgErrors.YamllError{Message: fmt.Sprintf("reading lock file errored with: '%v'", err)}
	}

	entries := make(map[string]LockEntry, len(lock.Entries))

	for _, entry := range lock.Entries {
		if entry.Source == "" {
			continue
		}

		entries[entry.Source] = entry
	}

	return entries, nil
}
