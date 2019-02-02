package commands

import (
	"github.com/molecule-man/stack-assembly/conf"
	"github.com/molecule-man/stack-assembly/stackassembly"
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

			// stacks, err := cfg.GetStacks()
			ss, err := cfg.StackConfigsSortedByExecOrder()
			handleError(err)

			cf := conf.Cf(cfg)

			for _, s := range ss {
				if len(args) > 0 && args[0] != s.Name {
					continue
				}

				printStackInfo(stackassembly.NewStack(cf, s.Name))
			}
		},
	}

	return infoCmd
}
