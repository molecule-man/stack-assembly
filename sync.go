package assembly

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/molecule-man/stack-assembly/awscf"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/conf"
)

func (sa SA) Sync(cfg conf.Config, nonInteractive bool) ([]*awscf.Stack, error) {
	return sa.syncRecursively(cfg, cfg, nonInteractive)
}

func (sa SA) syncRecursively(stackCfg conf.Config, root conf.Config, nonInteractive bool) ([]*awscf.Stack, error) {
	syncedStacks := []*awscf.Stack{}

	MustSucceed(stackCfg.Hooks.Pre.Exec())

	if stackCfg.Body != "" || stackCfg.UsePreviousTemplate {
		logger := sa.cli.PrefixedLogger(fmt.Sprintf("[%s] ", stackCfg.Name))

		logger.Info("Synchronizing template")

		stack, err := sa.exec(stackCfg, logger, nonInteractive)
		if err != nil {
			return syncedStacks, err
		}

		for _, r := range stackCfg.Blocked {
			logger.Infof("Blocking resource %s", r)

			err = stack.BlockResource(r)
			if err != nil {
				return syncedStacks, err
			}
		}

		syncedStacks = []*awscf.Stack{stack}
	}

	nestedStacks, err := stackCfg.StackConfigsSortedByExecOrder()
	if err != nil {
		return syncedStacks, err
	}

	for _, nestedStack := range nestedStacks {
		ss, err := sa.syncRecursively(nestedStack, root, nonInteractive)
		if err != nil {
			return syncedStacks, err
		}

		syncedStacks = append(syncedStacks, ss...)
	}

	return syncedStacks, stackCfg.Hooks.Post.Exec()
}

func (sa SA) exec(stackCfg conf.Config, logger *cli.Logger, nonInteractive bool) (*awscf.Stack, error) {
	cs := stackCfg.ChangeSet()

	chSet, err := sa.register(cs, logger)
	if errors.Is(err, awscf.ErrNoChange) {
		logger.Info("No changes to be synchronized")
		return cs.Stack(), nil
	}

	if err != nil {
		return cs.Stack(), err
	}

	defer func() {
		if closeErr := cs.Close(); closeErr != nil {
			logger.Warnf("Error while cleaning up: %s", closeErr.Error())
		}
	}()

	logger.Infof("Change set is created: %s", chSet.ID)

	sa.showChanges(chSet.Changes)

	if !nonInteractive {
		err = sa.letUserChooseNextAction(cs)
		if err != nil {
			return cs.Stack(), err
		}
	}

	if chSet.IsUpdate {
		err = stackCfg.Hooks.PreUpdate.Exec()
	} else {
		err = stackCfg.Hooks.PreCreate.Exec()
	}

	if err != nil {
		return cs.Stack(), err
	}

	wait := sa.showEvents(cs.Stack(), logger)

	err = chSet.Exec()

	wait <- true
	<-wait

	if err != nil {
		return cs.Stack(), err
	}

	if chSet.IsUpdate {
		err = stackCfg.Hooks.PostUpdate.Exec()
	} else {
		err = stackCfg.Hooks.PostCreate.Exec()
	}

	if err != nil {
		return cs.Stack(), err
	}

	logger.Print(sa.cli.Color.Success("Synchronization is complete"))

	return cs.Stack(), nil
}

func (sa SA) register(cs *awscf.ChangeSet, logger *cli.Logger) (*awscf.ChangeSetHandle, error) {
	chSet, err := cs.Register()

	if errors.Is(err, awscf.ErrStackAlreadyInProgress) {
		logger.Warn(err.Error())
		logger.Warn("Will wait until the current operation is complete")

		wait := sa.showEvents(cs.Stack(), logger)

		waitErr := cs.Stack().Wait()

		wait <- true
		<-wait

		if waitErr != nil {
			return chSet, waitErr
		}

		cs.Stack().Refresh()

		chSet, err = cs.Register()
	}

	var paramerr *awscf.ParametersMissingError

	if errors.As(err, &paramerr) {
		logger.Warn(paramerr.Error())

		for _, p := range paramerr.MissingParameters {
			response, rerr := sa.cli.Ask("Enter %s: ", p)
			MustSucceed(rerr)
			cs.WithParameter(p, response)
		}

		chSet, err = cs.Register()
	}

	return chSet, err
}

func (sa SA) showEvents(stack *awscf.Stack, logger *cli.Logger) chan bool {
	wait := make(chan bool)

	if _, err := stack.EventsTrack().FreshEvents(); err != nil {
		logger.Warnf("got an error while requesting stack events: %s", err)
	}

	go func() {
		writer := cli.NewColWriter(sa.cli.Writer, " ")

		for {
			events, err := stack.EventsTrack().FreshEvents()
			if err != nil {
				logger.Warnf("got an error while requesting stack events: %s", err)
			}

			for _, e := range events.Reversed() {
				logger.Fprint(writer, sa.sprintEvent(e))
			}

			writer.Flush()

			select {
			case <-wait:
				wait <- true
				return
			default:
				time.Sleep(2 * time.Second)
			}
		}
	}()

	return wait
}

func (sa SA) showChanges(changes []awscf.Change) {
	if len(changes) > 0 {
		t := cli.NewTable()
		t.Header("Action", "Resource Type", "Resource ID", "Replacement needed")

		for _, c := range changes {
			action := sa.cli.Color.Neutral(c.Action)

			switch strings.ToLower(c.Action) {
			case "add":
				action = sa.cli.Color.Success(c.Action)
			case "remove":
				action = sa.cli.Color.Fail(c.Action)
			}

			repl := sa.cli.Color.Success(fmt.Sprintf("%t", c.ReplacementNeeded))
			if c.ReplacementNeeded {
				repl = sa.cli.Color.Fail(fmt.Sprintf("%t", c.ReplacementNeeded))
			}

			t.Row(action, c.ResourceType, c.LogicalResourceID, repl)
		}

		sa.cli.Print(t.Render())
	}
}

func (sa SA) letUserChooseNextAction(chSet *awscf.ChangeSet) error {
	var actionErr error

	continueSync := false

	for !continueSync && actionErr == nil {
		err := sa.cli.Prompt([]cli.PromptCmd{
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
					diff, derr := awscf.ChSetDiff{Color: sa.cli.Color}.Diff(chSet)
					actionErr = derr

					if derr == nil {
						sa.cli.Print(diff)
					}
				},
			},
			{
				Description:   "[q]uit",
				TriggerInputs: []string{"q", "quit"},
				Action: func() {
					sa.cli.Error("Interrupted by user")
					actionErr = errors.New("sync is canceled")
				},
			},
		})
		if !errors.Is(err, cli.ErrPromptCommandIsNotKnown) {
			MustSucceed(err)
		}
	}

	return actionErr
}
