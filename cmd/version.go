package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"example.com/test/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of nearwait",
	Long:  `All software has versions. This is nearwait's`,
	Run: func(cmd *cobra.Command, args []string) {
		buildInfo := version.GetBuildInfo()
		fmt.Println(buildInfo)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
