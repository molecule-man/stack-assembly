package commands

import (
	fcolor "github.com/fatih/color"
	"github.com/molecule-man/stack-assembly/cli/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RootCmd() *cobra.Command {
	var nocolor bool
	rootCmd := &cobra.Command{
		Use: "stas",
	}
	rootCmd.PersistentFlags().StringP("profile", "p", "default", "AWS named profile")
	rootCmd.PersistentFlags().StringP("region", "r", "", "AWS region")

	rootCmd.PersistentFlags().StringSliceP("configs", "c", []string{},
		"Alternative config file(s). Default: stack-assembly.yaml")
	rootCmd.PersistentFlags().BoolVar(&nocolor, "nocolor", false,
		"Disables color output")

	rootCmd.AddCommand(infoCmd())
	rootCmd.AddCommand(syncCmd())
	rootCmd.AddCommand(diffCmd())
	rootCmd.AddCommand(deleteCmd())

	err := viper.BindPFlag("aws.profile", rootCmd.PersistentFlags().Lookup("profile"))
	handleError(err)

	err = viper.BindPFlag("aws.region", rootCmd.PersistentFlags().Lookup("region"))
	handleError(err)

	cobra.OnInitialize(func() {
		fcolor.NoColor = nocolor
		color.NoColor = nocolor
		err := viper.BindPFlags(rootCmd.PersistentFlags())
		handleError(err)
	})

	return rootCmd
}
