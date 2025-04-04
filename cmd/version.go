package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var (
	version = "unset"
)

func init() {
	c := &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(cmd *cobra.Command, args []string) {
			versionCmd()
		},
	}

	rootCmd.AddCommand(c)
}

func versionCmd() {
	fmt.Println(version)
}
