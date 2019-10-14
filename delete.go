package assembly

import (
	"errors"
	"fmt"

	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/conf"
)

func (sa *SA) Delete(cfg conf.Config, nonInteractive bool) error {
	action := &deleteAction{sa, sa.cli, nonInteractive}
	return action.delete(cfg)
}

type deleteAction struct {
	sa             *SA
	cli            *cli.CLI
	nonInteractive bool
}

func (a *deleteAction) delete(cfg conf.Config) error {
	ss, err := cfg.StackConfigsSortedByExecOrder()
	if err != nil {
		return err
	}

	// reverse order of stack configs
	for i, j := 0, len(ss)-1; i < j; i, j = i+1, j-1 {
		ss[i], ss[j] = ss[j], ss[i]
	}

	for _, s := range ss {
		err := a.delete(s)
		if err != nil {
			return err
		}
	}

	if cfg.Name == "" {
		return nil
	}

	logger := a.cli.PrefixedLogger(fmt.Sprintf("[%s] ", cfg.Name))

	stack := cfg.Stack()

	exists, err := stack.Exists()
	MustSucceed(err)

	if !exists {
		logger.Info("Stack doesn't exist")
		return nil
	}

	logger.Warnf("Stack %s is about to be deleted", cfg.Name)

	skip := false

	if !a.nonInteractive {
		continueDelete := false
		for !continueDelete && !skip {
			var actionErr error

			err = a.cli.Prompt([]cli.PromptCmd{{
				Description:   "[d]elete",
				TriggerInputs: []string{"d", "delete"},
				Action: func() {
					continueDelete = true
				},
			}, {
				Description:   "[a]ll (delete all without asking again)",
				TriggerInputs: []string{"a", "all"},
				Action: func() {
					a.nonInteractive = true
					continueDelete = true
				},
			}, {
				Description:   "[i]nfo (show stack info)",
				TriggerInputs: []string{"i", "info"},
				Action: func() {
					actionErr = a.sa.Info(stack)
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
					a.cli.Error("Interrupted by user")
					actionErr = errors.New("deletion is canceled")
				},
			}})
			if err != cli.ErrPromptCommandIsNotKnown {
				MustSucceed(err)
			}

			if actionErr != nil {
				return actionErr
			}
		}
	}

	if skip {
		return nil
	}

	err = stack.Delete()
	if err != nil {
		return err
	}

	logger.Print(a.cli.Color.Success("Stack is deleted successfully"))
	return nil
}
