package commands

import (
	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "stas",
	}
	rootCmd.PersistentFlags().StringSliceP("configs", "f", []string{}, "CF configs")

	rootCmd.AddCommand(infoCmd())
	rootCmd.AddCommand(syncCmd())

	return rootCmd
}
