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
	cfg := conf.Config{}

	cfg.Templates = map[string]conf.TemplateConfig{
		stackName: {
			Path: tpl,
			Name: stackName,
		},
	}

	sync(cfg)
}

func sync(cfg conf.Config) {
	serv := conf.InitStasService(cfg)

	tpls := make(map[string]stackassembly.StackTemplate)

	for i, template := range cfg.Templates {
		tplBody, err := ioutil.ReadFile(template.Path)
		handleError(err)

		tpls[i] = stackassembly.StackTemplate{
			Name:      template.Name,
			Body:      string(tplBody),
			Params:    template.Parameters,
			DependsOn: template.DependsOn,
			Blocked:   template.Blocked,
		}
	}

	err := serv.SyncAll(tpls, cfg.Parameters)
	handleError(err)
}
