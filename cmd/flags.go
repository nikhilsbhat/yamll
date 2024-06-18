package cmd

import (
	"github.com/spf13/cobra"
)

// Config holds the information of the cli config.
type Config struct {
	Files        []string
	NoValidation bool
	ToFile       string
}

// // Registers all global flags to utility.
func registerCommonFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&yamllCfg.LogLevel, "log-level", "", "INFO",
		"log level for the yamll")
	cmd.PersistentFlags().StringVarP(&cliCfg.ToFile, "to-file", "", "",
		"name of the file to which the final imported yaml should be written to")
	cmd.PersistentFlags().StringArrayVarP(&cliCfg.Files, "file", "f", nil,
		"root yaml files to be used for importing")
	cmd.PersistentFlags().BoolVarP(&cliCfg.NoValidation, "no-validation", "", false,
		"when enabled it skips validating the final generated yaml file")
}
