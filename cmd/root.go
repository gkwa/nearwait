package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/gkwa/nearwait/internal/logger"
)

var (
	cfgFile   string
	verbose   bool
	logFormat string
	cliLogger logr.Logger
)

var rootCmd = &cobra.Command{
	Use:   "nearwait",
	Short: "A brief description of your application",
	Long:  `A longer description that spans multiple lines and likely contains examples and usage of using your application.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize the console logger just before running
		// a command only if one wasn't provided. This allows other
		// callers (e.g. unit tests) to inject their own logger ahead of time.
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
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".nearwait")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

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
