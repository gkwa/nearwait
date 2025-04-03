package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/gkwa/nearwait/core"
	"github.com/gkwa/nearwait/internal/logger"
)

var (
	cfgFile      string
	verbose      bool
	logFormat    string
	cliLogger    logr.Logger
	force        bool
	debug        bool
	manifestFile string
	includes     []string
	noExclude    bool
	byteSize     int64
	prompt       bool
)

var rootCmd = &cobra.Command{
	Use:   "nearwait",
	Short: "Nearwait copies project files to clipboard based on a manifest",
	Long:  `Nearwait is a tool that copies project files to the clipboard according to what's specified in a local manifest YAML file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := LoggerFrom(cmd.Context())

		// Validate flags
		if prompt && byteSize == 0 {
			return fmt.Errorf("--prompt can only be used with --byte-size")
		}

		generator := core.NewManifestGenerator(logger)
		generator.WithFS(os.DirFS(".")) // Initialize with default filesystem
		if len(includes) > 0 {
			generator.WithIncludes(includes)
		}
		if noExclude {
			generator.DisableExcludes()
		}
		isNewManifest, err := generator.Generate(force, manifestFile)
		if err != nil {
			logger.Error(err, "Failed to generate manifest")
			return err
		}
		if isNewManifest {
			absPath, _ := filepath.Abs(manifestFile)
			fmt.Printf("%s generated successfully\n", absPath)
			return nil
		}
		processor := core.NewManifestProcessor(logger, debug, manifestFile)
		processor.WithByteSize(byteSize)
		processor.WithPrompt(prompt)
		isEmpty, err := processor.Process()
		if err != nil {
			logger.Error(err, "Failed to process manifest")
			return err
		}
		if isEmpty {
			absPath, _ := filepath.Abs(manifestFile)
			fmt.Fprintf(os.Stderr, "Manifest file list is empty from %s\n", absPath)
		}
		return nil
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cliLogger.IsZero() {
			cliLogger = logger.NewConsoleLogger(verbose, logFormat == "json")
		}
		ctx := logr.NewContext(context.Background(), cliLogger)
		cmd.SetContext(ctx)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.nearwait.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose mode")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "", "json or text (default is text)")
	rootCmd.PersistentFlags().BoolVar(&force, "force", false, "Force overwrite of existing manifest")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Keep temporary directory for debugging")
	rootCmd.PersistentFlags().StringVar(&manifestFile, "manifest", ".nearwait.yml", "Name of the manifest file")
	rootCmd.PersistentFlags().StringSliceVar(&includes, "include", nil, "Include only specified directories")
	rootCmd.PersistentFlags().BoolVar(&noExclude, "no-exclude", false, "Disable default directory exclusions")
	rootCmd.PersistentFlags().Int64Var(&byteSize, "byte-size", 0, "Maximum byte size per batch of files to copy")
	rootCmd.PersistentFlags().BoolVar(&prompt, "prompt", false, "Prompt before processing each batch (only valid with --byte-size)")

	if err := viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose")); err != nil {
		fmt.Printf("Error binding verbose flag: %v\n", err)
		os.Exit(1)
	}
	if err := viper.BindPFlag("log-format", rootCmd.PersistentFlags().Lookup("log-format")); err != nil {
		fmt.Printf("Error binding log-format flag: %v\n", err)
		os.Exit(1)
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".nearwait")
	}

	viper.AutomaticEnv()

	// Just read the config silently
	viper.ReadInConfig()

	logFormat = viper.GetString("log-format")
	verbose = viper.GetBool("verbose")
}

func LoggerFrom(ctx context.Context, keysAndValues ...interface{}) logr.Logger {
	if cliLogger.IsZero() {
		cliLogger = logger.NewConsoleLogger(verbose, logFormat == "json")
	}
	newLogger := cliLogger
	if ctx != nil {
		if l, err := logr.FromContext(ctx); err == nil {
			newLogger = l
		}
	}
	return newLogger.WithValues(keysAndValues...)
}
