package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func setCLIClient(_ *cobra.Command, _ []string) error {
	writer = os.Stdout

	if len(cliCfg.ToFile) != 0 {
		filePTR, err := os.Create(cliCfg.ToFile)
		if err != nil {
			return err
		}

		writer = filePTR
	}

	return nil
}
