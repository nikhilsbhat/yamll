package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

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
		RunE: func(cmd *cobra.Command, _ []string) error {
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
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg := yamll.New(yamllCfg.Merge, yamllCfg.LogLevel, yamllCfg.Limiter, cliCfg.Files...)
			cfg.SetLogger()
			logger = cfg.GetLogger()
			cfg.LockFile = cliCfg.LockFile
			cfg.NoLock = cliCfg.NoLock
			cfg.Profile = cliCfg.Profile

			out, err := cfg.Yaml()
			if err != nil {
				logger.Error("errored generating final yaml", slog.Any("err", err))
			}

			if !cliCfg.NoValidation {
				logger.Debug("validating final yaml for syntax")

				var data any

				if err = yaml.Unmarshal([]byte(out), &data); err != nil {
					logger.Error("the final rendered YAML file is not a valid yaml", slog.Any("error", err))
					logger.Error("rendering the final YAML encountered an error. skip validation to view the broken file.")

					os.Exit(1)
				}
			}

			if cliCfg.Explode {
				explodedOut, err := out.Explode()
				if err != nil {
					logger.Error("exploding final YAML errored", slog.Any("error", err))
					logger.Warn("rendering YAML without exploding, due to above errors")
				} else {
					out = explodedOut
				}
			}

			if !cliCfg.NoColor {
				render := renderer.GetRenderer(nil, nil, false, true, false, false, false)

				coloredFinalData, err := render.Color(renderer.TypeYAML, string(out))
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

func getBuildCommand() *cobra.Command {
	buildCommand := &cobra.Command{
		Use:     "build [flags]",
		Short:   "Builds YAML files substituting imports",
		Long:    "Builds YAML by substituting all anchors and aliases defined in sub-YAML files defined as libraries",
		Example: `yamll build --file path/to/file.yaml`,
		PreRunE: setCLIClient,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg := yamll.New(yamllCfg.Merge, yamllCfg.LogLevel, yamllCfg.Limiter, cliCfg.Files...)
			cfg.SetLogger()
			logger = cfg.GetLogger()
			cfg.LockFile = cliCfg.LockFile
			cfg.NoLock = cliCfg.NoLock
			cfg.Profile = cliCfg.Profile

			out, err := cfg.YamlBuild()
			if err != nil {
				logger.Error("errored generating final yaml", slog.Any("err", err))
			}

			if !cliCfg.NoValidation {
				validationStart := time.Now()

				logger.Debug("validating final yaml for syntax")

				var data any

				if err = yaml.Unmarshal([]byte(out), &data); err != nil {
					logger.Error("the final rendered YAML file is not a valid yaml", slog.Any("error", err))
					logger.Error("rendering the final YAML encountered an error. skip validation to view the broken file.")

					os.Exit(1)
				}

				if cfg.Profile {
					cfg.RecordValidationTiming(time.Since(validationStart))
				}
			}

			if !cliCfg.NoColor {
				render := renderer.GetRenderer(nil, nil, false, true, false, false, false)

				coloredFinalData, err := render.Color(renderer.TypeYAML, string(out))
				if err != nil {
					logger.Error("color coding yaml errored", slog.Any("error", err))
				} else {
					out = yamll.Yaml(coloredFinalData)
				}
			}

			if cliCfg.Profile {
				if _, err = fmt.Fprint(os.Stderr, cfg.ProfileReport()); err != nil {
					return err
				}

				return nil
			}

			if _, err = writer.Write([]byte(out)); err != nil {
				return err
			}

			return nil
		},
	}

	buildCommand.SilenceErrors = true
	registerCommonFlags(buildCommand)

	buildCommand.PersistentFlags().StringVarP(&cliCfg.ToFile, "to-file", "", "",
		"name of the file to which the final imported yaml should be written to")
	buildCommand.PersistentFlags().BoolVarP(&cliCfg.NoValidation, "no-validation", "", false,
		"when enabled it skips validating the final generated YAML file")
	buildCommand.PersistentFlags().BoolVarP(&cliCfg.Profile, "profile", "", false,
		"when enabled it prints timing information for build phases")

	return buildCommand
}

func getTreeCommand() *cobra.Command {
	treeCommand := &cobra.Command{
		Use:   "tree [flags]",
		Short: "Builds dependency trees from sub-YAML files defined as libraries",
		Long:  "Identifies dependencies and builds the dependency tree for the base yaml",
		Example: `yamll tree --file path/to/file.yaml
yamll tree --file path/to/file.yaml --output=json
yamll tree --file path/to/file.yaml --output=dot
yamll tree --file path/to/file.yaml --output=mermaid`,
		PreRunE: setCLIClient,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg := yamll.New(yamllCfg.Merge, yamllCfg.LogLevel, yamllCfg.Limiter, cliCfg.Files...)
			cfg.SetLogger()
			logger = cfg.GetLogger()
			cfg.LockFile = cliCfg.LockFile
			cfg.NoLock = cliCfg.NoLock

			out, err := cfg.Tree(cliCfg.TreeOutput, cliCfg.NoColor, cliCfg.ShowPattern)
			if err != nil {
				logger.Error("errored generating final yaml", slog.Any("err", err))
				os.Exit(1)
			}

			if _, err = writer.Write([]byte(out)); err != nil {
				return err
			}

			return nil
		},
	}

	treeCommand.SilenceErrors = true
	registerCommonFlags(treeCommand)
	treeCommand.PersistentFlags().StringVarP(&cliCfg.TreeOutput, "output", "o", yamll.TreeOutputText,
		"tree output format: text, json, dot, or mermaid")

	return treeCommand
}

func getTraceCommand() *cobra.Command {
	traceCommand := &cobra.Command{
		Use:   "trace [flags] <file:path|path>",
		Short: "Traces a generated YAML path back to its source file",
		Long:  "Traces a generated YAML path back to the source YAML file and line that produced it.",
		Example: `yamll trace internal/fixtures/import.yaml:base.movies
yamll trace --file internal/fixtures/import.yaml base.movies`,
		Args:    cobra.ExactArgs(1),
		PreRunE: setCLIClient,
		RunE: func(_ *cobra.Command, args []string) error {
			rootFile, tracePath := parseTraceTarget(args[0])
			if rootFile != "" {
				cliCfg.Files = []string{rootFile}
			}

			cfg := yamll.New(false, yamllCfg.LogLevel, yamllCfg.Limiter, cliCfg.Files...)
			cfg.SetLogger()
			logger = cfg.GetLogger()
			cfg.LockFile = cliCfg.LockFile
			cfg.NoLock = cliCfg.NoLock

			trace, err := cfg.Trace(tracePath)
			if err != nil {
				logger.Error("errored tracing yaml path", slog.Any("err", err))
				os.Exit(1)
			}

			if _, err = fmt.Fprintf(writer, "origin: %s\n", trace.Origin); err != nil {
				return err
			}

			return nil
		},
	}

	traceCommand.SilenceErrors = true
	registerCommonFlags(traceCommand)

	return traceCommand
}

func parseTraceTarget(target string) (string, string) {
	rootFile, tracePath, found := strings.Cut(target, ":")
	if !found {
		return "", target
	}

	return rootFile, tracePath
}

func getLockCommand() *cobra.Command {
	lockCommand := &cobra.Command{
		Use:     "lock [flags]",
		Short:   "Generates a lock file for reproducible remote imports",
		Long:    "Resolves remote imports and writes a lock file containing resolved commits and checksums.",
		Example: "yamll lock -f path/to/root.yaml",
		PreRunE: setCLIClient,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg := yamll.New(false, yamllCfg.LogLevel, yamllCfg.Limiter, cliCfg.Files...)
			cfg.SetLogger()

			logger = cfg.GetLogger()

			cfg.LockFile = cliCfg.LockFile
			cfg.NoLock = cliCfg.NoLock

			out, err := cfg.Lock()
			if err != nil {
				logger.Error("errored generating lock file", slog.Any("err", err))
				os.Exit(1)
			}

			lockPath := "yamll.lock"
			if cliCfg.ToFile != "" {
				lockPath = cliCfg.ToFile
			}

			const readPermission = 0o600

			if err = os.WriteFile(lockPath, out, readPermission); err != nil {
				return err
			}

			return nil
		},
	}

	lockCommand.SilenceErrors = true
	registerCommonFlags(lockCommand)

	return lockCommand
}

func getLintCommand() *cobra.Command {
	lintCommand := &cobra.Command{
		Use:     "lint [flags]",
		Short:   "Lints YAML imports/anchors/merges for common issues",
		Long:    "Runs static checks on the YAML import graph, anchors, and merge usage.",
		Example: "yamll lint -f path/to/root.yaml",
		PreRunE: setCLIClient,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg := yamll.New(false, yamllCfg.LogLevel, yamllCfg.Limiter, cliCfg.Files...)
			cfg.SetLogger()
			logger = cfg.GetLogger()
			cfg.LockFile = cliCfg.LockFile
			cfg.NoLock = cliCfg.NoLock

			report, err := cfg.Lint()
			if err != nil {
				logger.Error("lint errored", slog.Any("err", err))
				os.Exit(1)
			}

			for _, issue := range report.Issues {
				file := issue.File
				if file == "" {
					file = "-"
				}

				if _, err = fmt.Fprintf(writer, "%s\t%s\t%s\n", issue.Code, file, issue.Message); err != nil {
					return err
				}
			}

			if len(report.Issues) != 0 {
				os.Exit(1)
			}

			return nil
		},
	}

	lintCommand.SilenceErrors = true
	registerCommonFlags(lintCommand)

	return lintCommand
}

func versionConfig(_ *cobra.Command, _ []string) error {
	buildInfo, err := json.Marshal(version.GetBuildInfo())
	if err != nil {
		logger.Error("version fetch of yaml failed", slog.Any("err", err))
		os.Exit(1)
	}

	versionWriter := bufio.NewWriter(os.Stdout)
	versionInfo := fmt.Sprintf("%s \n", strings.Join([]string{"yamll version", string(buildInfo)}, ": "))

	if _, err = versionWriter.WriteString(versionInfo); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	defer func(writer *bufio.Writer) {
		err = writer.Flush()
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}
	}(versionWriter)

	return nil
}
