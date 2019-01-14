package commands

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "stas",
	}
	rootCmd.PersistentFlags().StringSliceP("configs", "c", []string{}, "CF configs")

	var nocolor bool
	rootCmd.PersistentFlags().BoolVar(&nocolor, "nocolor", false, "Disables color output")

	rootCmd.AddCommand(infoCmd())
	rootCmd.AddCommand(syncCmd())
	rootCmd.AddCommand(diffCmd())
	rootCmd.AddCommand(deleteCmd())

	cobra.OnInitialize(func() {
		color.NoColor = nocolor
	})

	return rootCmd
}
