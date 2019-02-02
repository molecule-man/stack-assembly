package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/cli/color"
	"github.com/molecule-man/stack-assembly/conf"
	"github.com/molecule-man/stack-assembly/stackassembly"
	"github.com/spf13/cobra"
)

func syncCmd() *cobra.Command {
	var nonInteractive bool

	syncCmd := &cobra.Command{
		Use:     "sync [ID]",
		Aliases: []string{"deploy"},
		Short:   "Synchronize (deploy) stacks",
		Long: `Creates or updates stacks specified in the config file(s).

By default sync command deploys all the stacks described in the config file(s).
To deploy a particular stack, ID argument has to be provided. ID is an
identifier of a stack within the config file. For example, ID is tpl1 in the
following yaml config:

    stacks:
      tpl1: # <--- this is ID
        name: mystack
        path: path/to/tpl.json`,

		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfgFiles, err := cmd.Parent().PersistentFlags().GetStringSlice("configs")
			handleError(err)

			cfg, err := conf.LoadConfig(cfgFiles)
			handleError(err)

			if len(args) > 0 {
				id := args[0]
				stack, ok := cfg.Stacks[id]
				if !ok {
					foundIds := make([]string, 0, len(cfg.Stacks))
					for id := range cfg.Stacks {
						foundIds = append(foundIds, id)
					}

					handleError(fmt.Errorf("ID %s is not found in the config. Found IDs: %v", id, foundIds))
				}
				cfg.Stacks = map[string]conf.StackConfig{
					id: stack,
				}
			}

			sync(cfg, nonInteractive)
		},
	}
	syncCmd.Flags().BoolVarP(&nonInteractive, "no-interaction", "n", false, "Do not ask any interactive questions")

	return syncCmd
}

func sync(cfg conf.Config, nonInteractive bool) {
	stackCfgs, err := cfg.StackConfigsSortedByExecOrder()
	handleError(err)

	handleError(cfg.Hooks.Pre.Exec())

	for _, stackCfg := range stackCfgs {
		logger := cli.PrefixedLogger(fmt.Sprintf("[%s] ", stackCfg.Name))

		logger.Info("Synchronizing template")

		cs, err := cfg.ChangeSetFromStackConfig(stackCfg)
		handleError(err)

		chSet, err := cs.Register()

		if paramerr, ok := err.(*stackassembly.ParametersMissingError); ok {
			logger.Warn(paramerr.Error())
			for _, p := range paramerr.MissingParameters {
				response, rerr := cli.Ask("Enter %s: ", p)
				handleError(rerr)
				cs.WithParameter(p, response)
			}

			chSet, err = cs.Register()
		}

		if err == stackassembly.ErrNoChange {
			logger.Info("No changes to be synchronized")
		} else {
			handleError(err)

			logger.Infof("Change set is created: %s", chSet.ID)

			showChanges(chSet.Changes)

			if !nonInteractive {
				letUserChooseNextAction(cs)
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

			events, stopTracking := et.StartTracking(cs.Stack())
			defer stopTracking()

			go func() {
				writer := cli.NewColWriter(cli.Output, " ")
				for e := range events {
					logger.Fprint(writer, sprintEvent(e))
					writer.Flush()
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
			logger.Print(color.Success("Synchronization is complete"))
		}

		for _, r := range stackCfg.Blocked {
			logger.Infof("Blocking resource %s", r)
			err := cs.Stack().BlockResource(r)

			handleError(err)
		}
	}

	handleError(cfg.Hooks.Post.Exec())
}

func showChanges(changes []stackassembly.Change) {
	if len(changes) > 0 {
		t := cli.NewTable()
		t.Header("Action", "Resource Type", "Resource ID", "Replacement needed")

		for _, c := range changes {
			action := color.Neutral(c.Action)
			switch strings.ToLower(c.Action) {
			case "add":
				action = color.Success(c.Action)
			case "remove":
				action = color.Fail(c.Action)
			}

			repl := color.Success(fmt.Sprintf("%t", c.ReplacementNeeded))
			if c.ReplacementNeeded {
				repl = color.Fail(fmt.Sprintf("%t", c.ReplacementNeeded))
			}
			t.Row(action, c.ResourceType, c.LogicalResourceID, repl)

		}

		cli.Print(t.Render())
	}
}

func letUserChooseNextAction(chSet *stackassembly.ChangeSet) {
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
					diff, derr := stackassembly.Diff(chSet)
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
