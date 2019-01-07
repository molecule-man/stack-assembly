package commands

import (
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/cmd/conf"
	"github.com/molecule-man/stack-assembly/stackassembly"
	"github.com/spf13/cobra"
)

func diffCmd() *cobra.Command {
	diffCmd := &cobra.Command{
		Use:   "diff [stack name]",
		Short: "Show diff of the stacks to be deployed",
		Run: func(cmd *cobra.Command, args []string) {
			cfgFiles, err := cmd.Parent().PersistentFlags().GetStringSlice("configs")
			handleError(err)

			cfg, err := conf.LoadConfig(cfgFiles)
			handleError(err)

			stacks, err := cfg.GetStacks()
			handleError(err)

			for _, stack := range stacks {
				diff, err := stackassembly.Diff(stack)
				handleError(err)

				cli.Print(diff)
			}
		},
	}

	return diffCmd
}
