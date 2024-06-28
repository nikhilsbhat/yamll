package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/nikhilsbhat/common/renderer"
	"github.com/nikhilsbhat/yamll/pkg/yamll"
	"github.com/nikhilsbhat/yamll/version"
	"github.com/spf13/cobra"
)

func getRootCommand() *cobra.Command {
	rootCommand := &cobra.Command{
		Use:   "yamll [command]",
		Short: "A utility to facilitate the inclusion of sub-YAML files as libraries.",
		Long:  `It identifies imports declared in YAML files and merges them to generate a single final YAML file, similar to importing libraries in programming.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Usage()
		},
	}
	rootCommand.SetUsageTemplate(getUsageTemplate())

	return rootCommand
}

func getVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version [flags]",
		Short: "Command to fetch the version of YAMLL installed",
		Long:  `This will help the user find what version of the YAMLL he or she installed in her machine.`,
		RunE:  versionConfig,
	}
}

func getImportCommand() *cobra.Command {
	importCommand := &cobra.Command{
		Use:   "import [flags]",
		Short: "Imports defined sub-YAML files as libraries",
		Long:  "Identifies dependency tree and imports them in the order to generate one single YAML file",
		Example: `yamll import --file path/to/file.yaml
yamll import --file path/to/file.yaml --no-validation
yamll import --file path/to/file.yaml --effective`,
		PreRunE: setCLIClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := yamll.New(yamllCfg.Effective, yamllCfg.LogLevel, yamllCfg.Limiter, cliCfg.Files...)
			cfg.SetLogger()
			logger = cfg.GetLogger()

			out, err := cfg.Yaml()
			if err != nil {
				logger.Error("errored generating final yaml", slog.Any("err", err))
			}

			if !cliCfg.NoValidation {
				logger.Debug("validating final yaml for syntax")
				var data interface{}
				err = yaml.Unmarshal([]byte(out), &data)
				if err != nil {
					logger.Error("the final rendered YAML file is not a valid yaml", slog.Any("error", err))
					logger.Error("rendering the final YAML encountered an error. skip validation to view the broken file.")

					os.Exit(1)
				}
			}

			if cliCfg.Explode {
				explodedOut, err := out.Explode(logger)
				if err != nil {
					logger.Error("exploding final YAML errored", slog.Any("error", err))
					logger.Warn("rendering YAML without exploding, due to above errors")
				} else {
					out = explodedOut
				}
			}

			if !cliCfg.NoColor {
				render := renderer.GetRenderer(nil, nil, false, true, false, false, false)
				coloredFinalData, err := render.ColorYAML(string(out))
				if err != nil {
					logger.Error("color coding yaml errored", slog.Any("error", err))
				} else {
					out = yamll.Yaml(coloredFinalData)
				}
			}

			if _, err = writer.Write([]byte(out)); err != nil {
				return err
			}

			return nil
		},
	}

	importCommand.SilenceErrors = true
	registerCommonFlags(importCommand)
	registerImportFlags(importCommand)

	return importCommand
}

func getTreeCommand() *cobra.Command {
	importCommand := &cobra.Command{
		Use:     "tree [flags]",
		Short:   "builds dependency trees from sub-YAML files defined as libraries",
		Long:    "Identifies dependencies and builds the dependency tree for the base yaml",
		Example: `yamll tree --file path/to/file.yaml`,
		PreRunE: setCLIClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := yamll.New(yamllCfg.Effective, yamllCfg.LogLevel, yamllCfg.Limiter, cliCfg.Files...)
			cfg.SetLogger()
			logger = cfg.GetLogger()

			if err := cfg.YamlTree(cliCfg.NoColor); err != nil {
				logger.Error("errored generating final yaml", slog.Any("err", err))
			}

			return nil
		},
	}

	importCommand.SilenceErrors = true
	registerCommonFlags(importCommand)

	return importCommand
}

func versionConfig(_ *cobra.Command, _ []string) error {
	buildInfo, err := json.Marshal(version.GetBuildInfo())
	if err != nil {
		logger.Error("version fetch of yaml failed", slog.Any("err", err))
		os.Exit(1)
	}

	writer := bufio.NewWriter(os.Stdout)
	versionInfo := fmt.Sprintf("%s \n", strings.Join([]string{"yamll version", string(buildInfo)}, ": "))

	if _, err = writer.WriteString(versionInfo); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	defer func(writer *bufio.Writer) {
		err = writer.Flush()
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}
	}(writer)

	return nil
}
