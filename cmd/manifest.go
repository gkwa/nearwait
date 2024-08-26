package cmd

import (
	"github.com/spf13/cobra"

	"github.com/gkwa/nearwait/core"
)

var (
	force        bool
	debug        bool
	manifestFile string
)

var manifestCmd = &cobra.Command{
	Use:   "manifest",
	Short: "Generate or process manifest",
	Long:  `Generate a new manifest or process an existing one.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger := LoggerFrom(cmd.Context())
		logger.Info("Running manifest command")

		generator := core.NewManifestGenerator(logger)
		if err := generator.Generate(force, manifestFile); err != nil {
			logger.Error(err, "Failed to generate manifest")
			return
		}

		processor := core.NewManifestProcessor(logger, debug, manifestFile)
		if err := processor.Process(); err != nil {
			logger.Error(err, "Failed to process manifest")
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(manifestCmd)
	manifestCmd.PersistentFlags().BoolVar(&force, "force", false, "Force overwrite of existing manifest")
	manifestCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Keep temporary directory for debugging")
	manifestCmd.PersistentFlags().StringVar(&manifestFile, "manifest", ".manifest.yml", "Name of the manifest file")
}
