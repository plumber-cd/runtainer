package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "dev"

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Long:  `All software has versions, right?`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("RunTainer " + Version)
	},
}
