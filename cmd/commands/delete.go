package commands

import (
	"fmt"

	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/cmd/conf"
	"github.com/molecule-man/stack-assembly/stackassembly"
	"github.com/spf13/cobra"
)

func deleteCmd() *cobra.Command {
	var nonInteractive bool

	cmd := &cobra.Command{
		Use:   "delete [stack name]",
		Short: "Deletes deployed stacks",
		Run: func(cmd *cobra.Command, args []string) {
			cfgFiles, err := cmd.Parent().PersistentFlags().GetStringSlice("configs")
			handleError(err)

			cfg, err := conf.LoadConfig(cfgFiles)
			handleError(err)

			ss, err := cfg.StackConfigsSortedByExecOrder()
			handleError(err)

			cf := conf.Cf(cfg)

			// reverse order of stack configs
			for i, j := 0, len(ss)-1; i < j; i, j = i+1, j-1 {
				ss[i], ss[j] = ss[j], ss[i]
			}

			for _, s := range ss {
				logger := cli.PrefixedLogger(fmt.Sprintf("[%s] ", s.Name))
				logger.Warn("Deleting stack")
				err := stackassembly.NewStack(cf, s.Name).Delete()
				handleError(err)
				logger.ColorPrint(cli.SuccessColor, "Stack is deleted successfully")
			}

		},
	}

	cmd.Flags().BoolVarP(&nonInteractive, "no-interaction", "n", false, "Do not ask any interactive questions")

	return cmd
}
