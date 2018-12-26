package commands

import (
	"io/ioutil"

	"github.com/molecule-man/stack-assembly/cmd/conf"
	"github.com/molecule-man/stack-assembly/stackassembly"
	"github.com/spf13/cobra"
)

func syncCmd() *cobra.Command {
	var stackName string

	syncCmd := &cobra.Command{
		Use:   "sync [tpl]",
		Short: "Sync stacks",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfgFiles, err := cmd.Parent().PersistentFlags().GetStringSlice("configs")
			handleError(err)

			if len(args) > 0 {
				execSyncOneTpl(stackName, args[0])
			} else {
				cfg, err := conf.LoadConfig(cfgFiles)
				handleError(err)
				sync(cfg)
			}
		},
	}
	syncCmd.Flags().StringVarP(&stackName, "stack", "s", "", "Stack name")

	return syncCmd
}

func execSyncOneTpl(stackName, tpl string) {
	cfg := stackassembly.Config{}

	cfg.Templates = map[string]stackassembly.StackTemplate{
		stackName: {
			Path: tpl,
			Name: stackName,
		},
	}

	sync(cfg)
}

func sync(cfg stackassembly.Config) {
	serv := conf.InitStasService(cfg)

	for i, template := range cfg.Templates {
		tplBody, err := ioutil.ReadFile(template.Path)
		handleError(err)

		template.Body = string(tplBody)
		cfg.Templates[i] = template
	}

	err := serv.Sync(cfg)
	handleError(err)
}
