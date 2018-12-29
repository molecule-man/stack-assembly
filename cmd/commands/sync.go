package commands

import (
	"io/ioutil"

	"github.com/molecule-man/stack-assembly/cmd/conf"
	"github.com/molecule-man/stack-assembly/stackassembly"
	"github.com/spf13/cobra"
)

func syncCmd() *cobra.Command {
	var stackName string
	var nonInteractive bool

	syncCmd := &cobra.Command{
		Use:   "sync [stack]",
		Short: "Sync stacks",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfgFiles, err := cmd.Parent().PersistentFlags().GetStringSlice("configs")
			handleError(err)

			if len(args) > 0 {
				execSyncOneTpl(stackName, args[0], nonInteractive)
			} else {
				cfg, err := conf.LoadConfig(cfgFiles)
				handleError(err)
				sync(cfg, nonInteractive)
			}
		},
	}
	syncCmd.Flags().StringVarP(&stackName, "stack", "s", "", "Stack name")
	syncCmd.Flags().BoolVarP(&nonInteractive, "no-interaction", "n", false, "Do not ask any interactive questions")

	return syncCmd
}

func execSyncOneTpl(stackName, tpl string, nonInteractive bool) {
	cfg := stackassembly.Config{}

	cfg.Stacks = map[string]stackassembly.StackConfig{
		stackName: {
			Path: tpl,
			Name: stackName,
		},
	}

	sync(cfg, nonInteractive)
}

func sync(cfg stackassembly.Config, nonInteractive bool) {
	serv := conf.InitStasService(cfg, nonInteractive)

	for i, stack := range cfg.Stacks {
		tplBody, err := ioutil.ReadFile(stack.Path)
		handleError(err)

		stack.Body = string(tplBody)
		cfg.Stacks[i] = stack
	}

	err := serv.Sync(cfg)
	handleError(err)
}
