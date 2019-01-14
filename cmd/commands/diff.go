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

			cc, err := cfg.ChangeSets()
			handleError(err)

			for _, c := range cc {
				diff, err := stackassembly.Diff(c)
				handleError(err)

				cli.Print(diff)
			}
		},
	}

	return diffCmd
}
