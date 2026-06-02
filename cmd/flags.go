package cmd

import (
	"github.com/spf13/cobra"
)

// Config holds the information of the cli config.
type Config struct {
	NoValidation bool
	Explode      bool
	NoColor      bool
	ShowPattern  bool
	LockFile     string
	NoLock       bool
	ToFile       string
	Files        []string
}

// Registers all global flags to utility.
func registerCommonFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&yamllCfg.LogLevel, "log-level", "l", "INFO",
		"log level for the yamll")
	cmd.PersistentFlags().StringVarP(&yamllCfg.Limiter, "limiter", "", "---",
		"limiters to separate the yaml files post merging")
	cmd.PersistentFlags().StringArrayVarP(&cliCfg.Files, "file", "f", nil,
		"root yaml files to be used for importing")
	cmd.PersistentFlags().BoolVarP(&cliCfg.NoColor, "no-color", "", false,
		"when enabled the output would not be color encoded")
	cmd.PersistentFlags().BoolVarP(&cliCfg.ShowPattern, "show-pattern-files", "", true,
		"when enabled, pattern imports in tree output will include matched filenames")
	cmd.PersistentFlags().StringVarP(&cliCfg.LockFile, "lock-file", "", "yamll.lock",
		"path to the lock file used for reproducible remote imports")
	cmd.PersistentFlags().BoolVarP(&cliCfg.NoLock, "no-lock", "", false,
		"when enabled, ignores any lock file during import/build/tree")
}

func registerImportFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&cliCfg.ToFile, "to-file", "", "",
		"name of the file to which the final imported yaml should be written to")
	cmd.PersistentFlags().BoolVarP(&cliCfg.NoValidation, "no-validation", "", false,
		"when enabled it skips validating the final generated YAML file")
	cmd.PersistentFlags().BoolVarP(&cliCfg.Explode, "explode", "", false,
		"when enabled, it expands any aliases and anchor tags present")
	cmd.PersistentFlags().BoolVarP(&yamllCfg.Merge, "merge", "", false,
		"when enabled it merges the yaml files effectively")

	cmd.MarkFlagsMutuallyExclusive("explode", "merge")
}
