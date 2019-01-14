package commands

import (
	"errors"
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

				stack := stackassembly.NewStack(cf, s.Name)

				exists, err := stack.Exists()
				handleError(err)

				if !exists {
					logger.Info("Stack doesn't exist")
				}

				logger.Warnf("Stack %s is about to be deleted", s.Name)

				skip := false

				if !nonInteractive {
					continueDelete := false
					for !continueDelete && !skip {
						err = cli.Prompt([]cli.PromptCmd{{
							Description:   "[d]elete",
							TriggerInputs: []string{"d", "delete"},
							Action: func() {
								continueDelete = true
							},
						}, {
							Description:   "[a]ll (delete all without asking again)",
							TriggerInputs: []string{"a", "all"},
							Action: func() {
								nonInteractive = true
								continueDelete = true
							},
						}, {
							Description:   "[i]nfo (show stack info)",
							TriggerInputs: []string{"i", "info"},
							Action: func() {
								printStackInfo(stack)
							},
						}, {
							Description:   "[s]kip",
							TriggerInputs: []string{"s", "skip"},
							Action: func() {
								skip = true
							},
						}, {
							Description:   "[q]uit",
							TriggerInputs: []string{"q", "quit"},
							Action: func() {
								cli.Error("Interrupted by user")
								handleError(errors.New("deletion is cancelled"))
							},
						}})
						if err != cli.ErrPromptCommandIsNotKnown {
							handleError(err)
						}
					}
				}

				if skip {
					continue
				}

				err = stack.Delete()
				handleError(err)
				logger.ColorPrint(cli.SuccessColor, "Stack is deleted successfully")
			}

		},
	}

	cmd.Flags().BoolVarP(&nonInteractive, "no-interaction", "n", false, "Do not ask any interactive questions")

	return cmd
}
