package commands

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
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
	cfg := conf.Config{}

	cfg.Stacks = map[string]stackassembly.StackConfig{
		stackName: {
			Path: tpl,
			Name: stackName,
		},
	}

	sync(cfg, nonInteractive)
}

func sync(cfg conf.Config, nonInteractive bool) {
	stackCfgs, err := cfg.StackConfigsSortedByExecOrder()
	handleError(err)

	logger := log.New(os.Stderr, "", log.LstdFlags)

	handleError(cfg.Hooks.Pre.Exec())

	for _, stackCfg := range stackCfgs {
		stack, err := cfg.NewStack(stackCfg)
		handleError(err)

		print := func(msg string, args ...interface{}) {
			logger.Print(fmt.Sprintf(fmt.Sprintf("[%s] %s", stack.Name, msg), args...))
		}

		print("Synchronizing template")

		chSet, err := stack.ChangeSet()

		if paramerr, ok := err.(*stackassembly.ParametersMissingError); ok {
			c := color.New(color.FgYellow, color.Bold)
			print(c.Sprint(paramerr.Error()))
			for _, p := range paramerr.MissingParameters {
				response, rerr := cli.Ask("Enter %s: ", p)
				handleError(rerr)
				stack.AddParameter(p, response)
			}

			chSet, err = stack.ChangeSet()
		}

		if err == stackassembly.ErrNoChange {
			print("No changes to be synced")
		} else {
			handleError(err)

			print("Change set is created: %s", chSet.ID)

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
					print(sprintEvent(e))
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
			print("Synchronization is finished")
		}

		for _, r := range stackCfg.Blocked {
			print("Blocking resource %s", r)
			err := stack.BlockResource(r)

			handleError(err)
		}
	}

	handleError(cfg.Hooks.Post.Exec())
}

func showChanges(changes []stackassembly.Change) {
	if len(changes) > 0 {
		t := cli.NewTable()
		t.Header().Cell("Action").Cell("ResourceType").Cell("Resource ID").Cell("Replacement needed")

		for _, c := range changes {
			t.Row()

			switch strings.ToLower(c.Action) {
			case "add":
				t.ColorizedCell(c.Action, green)
			case "remove":
				t.ColorizedCell(c.Action, boldRed)
			default:
				t.ColorizedCell(c.Action, cyan)
			}

			t.Cell(c.ResourceType)
			t.Cell(c.LogicalResourceID)

			col := green
			if c.ReplacementNeeded {
				col = boldRed
			}
			t.ColorizedCell(fmt.Sprintf("%t", c.ReplacementNeeded), col)
		}

		fmt.Println(t.Render())
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

					fmt.Println(diff)
				},
			},
			{
				Description:   "[q]uit",
				TriggerInputs: []string{"q", "quit"},
				Action: func() {
					print("Interrupted by user")
					handleError(errors.New("sync is cancelled"))
				},
			},
		})
		if err != cli.ErrPromptCommandIsNotKnown {
			handleError(err)
		}
	}
}
