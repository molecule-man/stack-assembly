package commands

import (
	"os"

	"github.com/molecule-man/stack-assembly/cmd/conf"
	"github.com/spf13/cobra"
)

func infoCmd() *cobra.Command {
	infoCmd := &cobra.Command{
		Use:   "info [stack name]",
		Short: "Show info about the stack",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfgFiles, err := cmd.Parent().PersistentFlags().GetStringSlice("configs")
			handleError(err)

			cfg, err := conf.LoadConfig(cfgFiles)
			handleError(err)

			printer := newInfoPrinter(os.Stdout, conf.Aws(cfg))

			stacks, err := cfg.GetStacks()
			handleError(err)

			for _, stack := range stacks {
				if len(args) > 0 && args[0] != stack.Name && args[0] != stack.ID {
					continue
				}

				printer.print(stack)
			}
		},
	}

	return infoCmd
}
