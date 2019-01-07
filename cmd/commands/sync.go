package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/molecule-man/stack-assembly/cli"
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
				cfg := conf.Config{}

				cfg.Stacks = map[string]stackassembly.StackConfig{
					stackName: {
						Path: args[0],
						Name: stackName,
					},
				}

				sync(cfg, nonInteractive)
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

func sync(cfg conf.Config, nonInteractive bool) {
	stackCfgs, err := cfg.StackConfigsSortedByExecOrder()
	handleError(err)

	handleError(cfg.Hooks.Pre.Exec())

	for _, stackCfg := range stackCfgs {
		stack, err := cfg.NewStack(stackCfg)
		handleError(err)

		logger := cli.PrefixedLogger(fmt.Sprintf("[%s] ", stack.Name))

		logger.Info("Synchronizing template")

		chSet, err := stack.ChangeSet()

		if paramerr, ok := err.(*stackassembly.ParametersMissingError); ok {
			logger.Warn(paramerr.Error())
			for _, p := range paramerr.MissingParameters {
				response, rerr := cli.Ask("Enter %s: ", p)
				handleError(rerr)
				stack.AddParameter(p, response)
			}

			chSet, err = stack.ChangeSet()
		}

		if err == stackassembly.ErrNoChange {
			logger.Info("No changes to be synchronized")
		} else {
			handleError(err)

			logger.Infof("Change set is created: %s", chSet.ID)

			showChanges(chSet.Changes)

			if !nonInteractive {
				letUserChooseNextAction(stack)
			}

			handleError(cfg.Hooks.PreSync.Exec())
			handleError(stackCfg.Hooks.PreSync.Exec())

			if chSet.IsUpdate {
				handleError(cfg.Hooks.PreUpdate.Exec())
				handleError(stackCfg.Hooks.PreUpdate.Exec())
			} else {
				handleError(cfg.Hooks.PreCreate.Exec())
				handleError(stackCfg.Hooks.PreCreate.Exec())
			}

			et := stackassembly.EventsTracker{}

			events, stopTracking := et.StartTracking(&stack)
			defer stopTracking()

			go func() {
				for e := range events {
					logger.Info(sprintEvent(e))
				}
			}()

			err = chSet.Exec()
			handleError(err)

			handleError(cfg.Hooks.PostSync.Exec())
			handleError(stackCfg.Hooks.PostSync.Exec())

			if chSet.IsUpdate {
				handleError(cfg.Hooks.PostUpdate.Exec())
				handleError(stackCfg.Hooks.PostUpdate.Exec())
			} else {
				handleError(cfg.Hooks.PostCreate.Exec())
				handleError(stackCfg.Hooks.PostCreate.Exec())
			}
			logger.ColorPrint(cli.SuccessColor, "Synchronization is complete")
		}

		for _, r := range stackCfg.Blocked {
			logger.Infof("Blocking resource %s", r)
			err := stack.BlockResource(r)

			handleError(err)
		}
	}

	handleError(cfg.Hooks.Post.Exec())
}

func showChanges(changes []stackassembly.Change) {
	if len(changes) > 0 {
		t := cli.NewTable()
		t.Header().Cell("Action").Cell("Resource Type").Cell("Resource ID").Cell("Replacement needed")

		for _, c := range changes {
			t.Row()

			switch strings.ToLower(c.Action) {
			case "add":
				t.ColorizedCell(c.Action, cli.SuccessColor)
			case "remove":
				t.ColorizedCell(c.Action, cli.FailureColor)
			default:
				t.ColorizedCell(c.Action, cli.NeutralColor)
			}

			t.Cell(c.ResourceType)
			t.Cell(c.LogicalResourceID)

			col := cli.SuccessColor
			if c.ReplacementNeeded {
				col = cli.FailureColor
			}
			t.ColorizedCell(fmt.Sprintf("%t", c.ReplacementNeeded), col)
		}

		cli.Print(t.Render())
	}
}

func letUserChooseNextAction(stack stackassembly.Stack) {
	continueSync := false
	for !continueSync {
		err := cli.Prompt([]cli.PromptCmd{
			{
				Description:   "[s]ync",
				TriggerInputs: []string{"s", "sync"},
				Action: func() {
					continueSync = true
				},
			},
			{
				Description:   "[d]iff",
				TriggerInputs: []string{"d", "diff"},
				Action: func() {
					diff, derr := stackassembly.Diff(stack)
					handleError(derr)

					cli.Print(diff)
				},
			},
			{
				Description:   "[q]uit",
				TriggerInputs: []string{"q", "quit"},
				Action: func() {
					cli.Error("Interrupted by user")
					handleError(errors.New("sync is cancelled"))
				},
			},
		})
		if err != cli.ErrPromptCommandIsNotKnown {
			handleError(err)
		}
	}
}
